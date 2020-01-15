package cproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	llog "log"
	"net"
	"net/http"
	"net/url"
	"os"
)

var log *llog.Logger = llog.New(os.Stdout, "", llog.LstdFlags|llog.Lshortfile)

type FakeServer struct {
	host     string
	address  string
	upgrader *websocket.Upgrader
	isTls    bool
	handler  *ProxyHander
}

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

func (p *FakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if IsWebSocketRequest(r) {
		wsConn, err := defaultUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ws upgrade error", err)
			return
		}
		defer wsConn.Close()

		newHeader := make(http.Header)
		for k, vs := range r.Header {
			switch {
			case k == "Upgrade" ||
				k == "Connection" ||
				k == "Sec-Websocket-Key" ||
				k == "Sec-Websocket-Version" ||
				k == "Sec-Websocket-Extensions":
			default:
				newHeader[k] = vs
			}
		}

		u := url.URL{Scheme: "ws", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
		if p.isTls {
			u.Scheme = "wss"
			websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}

		websocket.DefaultDialer.Proxy = http.ProxyURL(p.handler.Proxy.GetProxy())

		proxy, _, err := websocket.DefaultDialer.Dial(u.String(), newHeader)
		//dumpResp(resp)
		if err != nil {
			log.Println("client ws upgrade error", err)
			return
		}
		defer proxy.Close()
		// handle Websocket request
		conn := NewConn(wsConn)
		conn.AfterReadFunc = func(messageType int, data string) {
			if err := proxy.WriteMessage(websocket.TextMessage, []byte(data)); err != nil {
				log.Println("write to server error", err)
			}
		}
		go func() {
			for {
				_, message, err := proxy.ReadMessage()
				if err != nil {
					log.Println("read from server error:", err)
					return
				}
				if p.handler.Proxy.BeforeWsResponse(conn, r, message){
					continue
				}
				if _, e := conn.Write(message); e != nil {
					log.Println("ws write to client:", e)
				}
			}
		}()
		conn.Listen()

	} else {
		handleHttp(p.handler, w, r, true)
	}
}

func NewFakeServer(host string, p *ProxyHander) (int, *http.Server) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	domain, port := getSplitHostPort(host)
	h := &FakeServer{
		host:    domain,
		address: host,
		handler: p,
	}
	server := &http.Server{Handler: h}
	if port == "443" {
		h.isTls = true
		caPub, caPriv := GetCAPairPath(domain)
		go server.ServeTLS(listener, caPub, caPriv)
	} else {
		go server.Serve(listener)
	}
	return listener.Addr().(*net.TCPAddr).Port, server
}

type ProxyHander struct {
	Proxy *Proxy
}

func (p *ProxyHander) handleConnect(w http.ResponseWriter, r *http.Request) {
	port, server := NewFakeServer(r.Host, p)
	defer server.Shutdown(context.Background())
	proxyClient, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Println("---", r.Host, err, "---")
		return
	}
	fmt.Fprint(w, "HTTP/1.1 200 Connection established\r\n\r\n")
	hij, ok := w.(http.Hijacker)
	if !ok {
		panic("httpserver does not support hijacking")
	}

	realClient, _, e := hij.Hijack()
	if e != nil {
		panic("Cannot hijack connection " + e.Error())
	}
	defer realClient.Close()
	defer proxyClient.Close()
	go io.Copy(realClient, proxyClient)
	io.Copy(proxyClient, realClient)
}

func handleHttp(p *ProxyHander, w http.ResponseWriter, r *http.Request, ssl bool) {
	tr := &http.Transport{}
	scheme := "http"
	if ssl {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		scheme = "https"
	}
	if p.Proxy.GetProxy() != nil {
		tr.Proxy = http.ProxyURL(p.Proxy.GetProxy())
	}

	client := &http.Client{Transport: tr }
	newUrl := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())

	newReq, err := http.NewRequest(r.Method, newUrl, r.Body)
	if err != nil {
		log.Println("creat new request error", err)
		return
	}
	newReq.Header = r.Header
	newReq.Host = r.Host
	newReq.Header.Add("Host", r.Host)
	delHopHeaders(newReq.Header)

	var save io.ReadCloser
	save, newReq.Body, _ = drainBody(newReq.Body)
	resp, err := client.Do(newReq)
	if err != nil {
		llog.Printf("request error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	newReq.Body = save

	if p.Proxy.BeforeResponse(w, newReq, resp) {
		return
	}
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ProxyHander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		p.handleConnect(w, r)
	} else {
		handleHttp(p, w, r, false)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

var hopHeaders = []string{
	"Proxy-Connection",
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}
