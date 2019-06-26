package httpu

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

func GetUrl(r *http.Request) string {
	URL, _ := url.ParseRequestURI(r.RequestURI)
	URL.Host = r.Host
	URL.Scheme = r.URL.Scheme
	return URL.String()
}

func URLScheme(r *http.Request, scheme string, pth ...string) string {
	return scheme + "://" + r.Host + "/" + strings.TrimPrefix(path.Join(pth...), "/")
}

func URL(r *http.Request, pth ...string) string {
	return URLScheme(r, HttpScheme(r), pth...)
}

func WsURL(r *http.Request, pth ...string) string {
	return URLScheme(r, WsScheme(r), pth...)
}

func HttpScheme(r *http.Request) (scheme string) {
	if scheme = r.Header.Get("X-Forwarded-Proto"); scheme == "" {
		if r.TLS != nil {
			return "https"
		}
	}
	if scheme == "https" {
		return
	}
	return "http"
}

func WsScheme(r *http.Request) (scheme string) {
	if scheme = r.Header.Get("X-Forwarded-Proto"); scheme == "" {
		if r.TLS != nil {
			return "wss"
		}
		return "ws"
	}
	if scheme == "https" {
		return "wss"
	}
	return "ws"
}
