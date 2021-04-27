package httpu

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	HeaderXRequestedWith  = "X-Requested-With"
	HeaderXRequestedFrame = "X-Requested-Frame"

	RequestedByActionFormFrame = "Action"
)

func StripPrefix(w http.ResponseWriter, r *http.Request, handler http.Handler, prefix string, slashPermanentRedirect bool) {
	if prefix == "" {
		handler.ServeHTTP(w, r)
		return
	}

	if (r.URL.Path + "/") == prefix {
		if slashPermanentRedirect {
			http.Redirect(w, r, prefix, http.StatusPermanentRedirect)
		} else {
			http.Redirect(w, r, prefix, http.StatusTemporaryRedirect)
		}
		return
	}

	if prefix != "/" {
		if p := "/" + strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r = r.WithContext(context.WithValue(r.Context(), CtxPrefix, prefix))
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			handler.ServeHTTP(w, r2)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	handler.ServeHTTP(w, r)
}

func RemoteIP(r *http.Request) (ip net.IP) {
	if strings.ContainsRune(r.RemoteAddr, ':') {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		return net.ParseIP(host)
	}
	return net.ParseIP(r.RemoteAddr)
}

func IsActionFormRequest(r *http.Request) bool {
	return r.Header.Get(HeaderXRequestedFrame) == RequestedByActionFormFrame
}

func IsXhrRequest(r *http.Request) bool {
	return r.Header.Get(HeaderXRequestedWith) == "XMLHttpRequest"
}

func Redirect(w http.ResponseWriter, r *http.Request, url string, status int, force ...bool) {
	RedirectHeader("X-Location", w, r, url, status, force...)
}

func RedirectHeader(headerName string, w http.ResponseWriter, r *http.Request, url string, status int, force ...bool) {
	var force_ bool
	for _, force_ = range force{}

	if IsActionFormRequest(r) {
		if status < 400 {
			status = 201
		}

		w.Header().Set(headerName, url)
		w.WriteHeader(status)
	} else if IsXhrRequest(r) {
		if r.Header.Get("X-Redirection-Disabled") != "true" {
			w.Header().Set(headerName, url)
		} else if force_ {
			w.Header().Set(headerName+"-Window", url)
		}
		if status < 400 {
			status = 201
		}
		w.WriteHeader(status)
	} else {
		http.Redirect(w, r, url, status)
	}
}

// StripStaticPrefix returns a handler that serves HTTP requests
// by removing the given prefix from the request URL's Path
// and invoking the handler h.
func StripStaticPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			h.ServeHTTP(w, r2)
		}
	})
}
