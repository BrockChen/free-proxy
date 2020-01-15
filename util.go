package cproxy

import (
	"net/http"
	"os"
	"path"
	"strings"
)

func IsWebSocketRequest(r *http.Request) bool {
	contains := func(key, val string) bool {
		vv := strings.Split(r.Header.Get(key), ",")
		for _, v := range vv {
			if val == strings.ToLower(strings.TrimSpace(v)) {
				return true
			}
		}
		return false
	}
	if !contains("Connection", "upgrade") {
		return false
	}
	if !contains("Upgrade", "websocket") {
		return false
	}
	return true
}

func getSplitHostPort(host string) (ip, port string) {
	sps := strings.Split(host, ":")
	if len(sps) > 1 {
		return sps[0], sps[1]
	}
	return host, ""
}

func WriteToFile(filePath string, data []byte) error {
	err := os.MkdirAll(path.Dir(filePath), os.ModePerm)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	f.Write(data)
	f.Close()
	return nil
}