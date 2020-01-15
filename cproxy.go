package cproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var (
	LEVEL_0 = 0
	LEVEL_1 = 1
	LEVEL_2 = 2
	LEVEL_3 = 3
)

type Proxy struct {
	Proxy       *url.URL
	RedisPool   *redis.Pool
	Regexp      RuleOperator
	BindAddr    string
	proxyHander *ProxyHander
	Level       int
}

type Message struct {
	Url         string            `json:"url"`
	Method      string            `json:"method"`
	ReqHeader   map[string]string `json:"req-header"`
	RespHeader  map[string]string `json:"rsp-header"`
	ReqContent  string            `json:"req"`
	RespContent string            `json:"resp"`
	Status      int               `json:"status"`
}

type MessageReq struct {
	Url     string            `json:"url"`
	Method  string            `json:"method"`
	Header  map[string]string `json:"req-header"`
	Content []byte            `json:"req"`
}
type MessageResp struct {
	Status  int `json:"status"`
	Header  map[string]string
	Content []byte `json:"resp"`
}

func NewProxy(bindaddr, redisUri, rulePath, proxyUri, filter string) *Proxy {
	p := &Proxy{
		proxyHander: &ProxyHander{},
		Proxy:       nil,
		RedisPool:   nil,
		Level:       LEVEL_1,
		BindAddr:    bindaddr,
	}
	p.proxyHander.Proxy = p
	p.initProxy(proxyUri)
	p.initRedis(redisUri)
	p.initRules(rulePath, filter)
	return p
}

func (p *Proxy) BeforeResponse(w http.ResponseWriter, req *http.Request, resp *http.Response) (ret bool) {
	ret = false
	rule, ok := p.Regexp.Match(req.Host, req.URL.RequestURI())
	if !ok && p.Regexp.Enable {
		//if !ok {
		return
	}
	reqMsg := p.dumpReq(req)
	respMsg := p.dumpResp(resp)
	if reqMsg == nil || respMsg == nil {
		return false
	}

	if p.Level > LEVEL_0 {
		fmt.Printf("---------------\n")
		fmt.Printf("> %s %s\n", reqMsg.Method, reqMsg.Url)
	}
	if p.Level > LEVEL_1 {
		for k, v := range reqMsg.Header {
			fmt.Printf("> %s: %s\n", k, v)
		}
	}
	if p.Level > LEVEL_2 {
		fmt.Printf("\n> %s\n", reqMsg.Content)
	}

	if p.Level > LEVEL_0 {
		fmt.Printf("\n< %d \n", respMsg.Status)
	}
	if p.Level > LEVEL_1 {
		for k, v := range respMsg.Header {
			fmt.Printf("< %s: %s\n", k, v)
		}
	}
	if p.Level > LEVEL_2 {
		fmt.Printf("\n< %s\n", respMsg.Content)
	}

	if rule.Option == OPT_USE_LOCAL_RESPONSE {
		f, err := os.OpenFile(rule.Content, os.O_RDONLY, 0666)
		if err != nil {
			return
		}
		defer f.Close()

		copyHeader(w.Header(), resp.Header)
		w.Header().Del("Content-Length")
		w.WriteHeader(resp.StatusCode)

		io.Copy(w, f)
		ret = true
		return
	}

	if rule.Option == OPT_TO_REDIS {
		m := Message{
			Url:         reqMsg.Url,
			Method:      reqMsg.Method,
			ReqHeader:   reqMsg.Header,
			RespHeader:  respMsg.Header,
			ReqContent:  base64.StdEncoding.EncodeToString(reqMsg.Content),
			RespContent: base64.StdEncoding.EncodeToString(respMsg.Content),
			Status:      respMsg.Status,
		}
		if msg, e := json.Marshal(&m); e == nil {
			if p.RedisPool != nil {
				p.RedisPool.Get().Do("lpush", "http-message-queue", msg)
			} else {
				fmt.Printf("no redis found")
			}
		}
	}
	return false
}
func (p *Proxy) BeforeWsResponse(w *Conn, req *http.Request, message []byte) (ret bool) {
	ret = false
	rule, ok := p.Regexp.Match(req.Host, req.URL.RequestURI())
	if !ok && p.Regexp.Enable {
		return
	}
	if p.Level > LEVEL_0 {
		fmt.Printf("---------------\n")
		fmt.Printf("> %s %s\n", req.Method, req.URL.RequestURI())
	}

	if p.Level > LEVEL_1 {
		for k, v := range req.Header {
			fmt.Printf("> %s: %s\n", k, v[0])
		}
	}

	if rule.Option == OPT_USE_LOCAL_RESPONSE {
		f, err := os.OpenFile(rule.Content, os.O_RDONLY, 0666)
		if err != nil {
			return
		}
		defer f.Close()

		io.Copy(w, f)
		ret = true
		return
	}

	if rule.Option == OPT_TO_REDIS {
		h := make(map[string]string, 0)
		for k, v := range req.Header {
			h[k] = v[0]
		}
		m := Message{
			Url:         req.URL.RequestURI(),
			Method:      req.Method,
			ReqHeader:   h,
			RespContent: base64.StdEncoding.EncodeToString(message),
			Status:      206,
		}
		if msg, e := json.Marshal(&m); e == nil {
			if p.RedisPool != nil {
				p.RedisPool.Get().Do("lpush", "http-message-queue", msg)
			} else {
				fmt.Printf("no redis found")
			}
		}
	}
	return false
}

func (p *Proxy) GetProxy() *url.URL {
	return p.Proxy
}
func (p *Proxy) GetRedisConnection() redis.Conn {
	if p.RedisPool != nil {
		return p.RedisPool.Get()
	}
	return nil
}

func (p *Proxy) Run() error {
	fmt.Printf("proxy listen on %s\n", p.BindAddr)
	return http.ListenAndServe(p.BindAddr, p.proxyHander)
}

func (p *Proxy) dumpReq(req *http.Request) (msg *MessageReq) {
	message := &MessageReq{}
	var err error
	save := req.Body
	if req.Body != nil {
		save, req.Body, err = drainBody(req.Body)
		if err != nil {
			return
		}
	}
	reqURI := req.URL.RequestURI()
	message.Method = valueOrDefault(req.Method, "GET")
	message.Url = reqURI

	message.Header = make(map[string]string)
	absRequestURI := strings.HasPrefix(req.RequestURI, "http://") || strings.HasPrefix(req.RequestURI, "https://")
	if !absRequestURI {
		host := req.Host
		if host == "" && req.URL != nil {
			host = req.URL.Host
		}
		if host != "" {
			message.Header["Host"] = host
		}
	}

	chunked := len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked"
	if len(req.TransferEncoding) > 0 {
		//fmt.Fprintf(&b, "Transfer-Encoding: %s\r\n", strings.Join(req.TransferEncoding, ","))
		message.Header["Transfer-Encoding"] = strings.Join(req.TransferEncoding, ",")
	}
	if req.Close {
		message.Header["Connection"] = "close"
	}
	message.Header = make(map[string]string)
	for k, v := range req.Header {
		message.Header[k] = v[0]
	}
	var b bytes.Buffer
	if req.Body != nil {
		var dest io.Writer = &b
		if chunked {
			dest = httputil.NewChunkedWriter(dest)
		}
		_, err = io.Copy(dest, req.Body)
		if chunked {
			dest.(io.Closer).Close()
		}
	}
	message.Content = b.Bytes()
	req.Body = save
	return message
}
func (p *Proxy) dumpResp(resp *http.Response) (msg *MessageResp) {
	message := &MessageResp{}
	message.Status = resp.StatusCode
	message.Header = make(map[string]string)

	recordBody := false
	for k, v := range resp.Header {
		message.Header[k] = v[0]
		if k == "content-type" || k == "Content-Type" {
			if strings.Contains(v[0], "text") {
				recordBody = true
			}
		}

	}
	if !recordBody {
		return message
	}
	var b bytes.Buffer
	var err error
	save := resp.Body
	savecl := resp.ContentLength

	// For content length of zero. Make sure the body is an empty
	// reader, instead of returning error through failureToReadBody{}.
	//if resp.ContentLength <= 0 {
	//	return message
	//}

	if resp.Body == nil {
		resp.Body = emptyBody
	} else {
		save, resp.Body, err = drainBody(resp.Body)
		if err != nil {
			return
		}
	}
	_, err = io.Copy(&b, resp.Body)
	//err = resp.Write(&b)
	//if err == errNoBody {
	//	err = nil
	//}
	resp.Body = save
	resp.ContentLength = savecl
	if err != nil {
		return
	}
	message.Content = b.Bytes()
	return message
}

func (p *Proxy) initRedis(uri string) {
	if len(uri) == 0 {
		return
	}
	p.RedisPool = &redis.Pool{
		MaxIdle:     16,
		MaxActive:   0,
		IdleTimeout: 300,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", uri)
		},
	}
}
func (p *Proxy) initRules(ruleFilePath, filter string) {
	p.Regexp = NewRuleOperator(ruleFilePath, filter)

}
func (p *Proxy) initProxy(uri string) {
	if len(uri) == 0 {
		return
	}
	if u, e := url.Parse(uri); e == nil {
		p.Proxy = u
	} else {
		panic("init proxy")
	}
}

var reqWriteExcludeHeaderDump = map[string]bool{
	"Host":              true, // not in Header map anyway
	"Transfer-Encoding": true,
	"Trailer":           true,
}

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {

	if b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func valueOrDefault(value, def string) string {
	if value != "" {
		return value
	}
	return def
}

var emptyBody = ioutil.NopCloser(strings.NewReader(""))
var errNoBody = errors.New("sentinel error value")
