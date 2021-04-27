package helpers

import (
	"net/http"
	"strings"
)

func ReadUserIP(r *http.Request) string {
	addr := r.Header.Get("X-Real-Ip")
	if addr == "" {
		if addr = r.Header.Get("X-Forwarded-For"); addr != "" {
			addr = strings.TrimSpace(strings.Split(addr, ",")[0])
		}
	}
	if addr == "" {
		addr = r.RemoteAddr
	}
	return addr
}
