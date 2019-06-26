package middlewares

import "net/http"

const HeaderRemoteAddr = "X-Real-IP"

func RemoteAddrHeaderMiddleware(allowedFrom func(r *http.Request) bool) func(handler http.Handler) http.Handler {
	if allowedFrom == nil {
		return func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if realIp := r.Header.Get(HeaderRemoteAddr); realIp != "" {
					r.RemoteAddr = realIp
				}
				handler.ServeHTTP(w, r)
			})
		}
	}
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if realIp := r.Header.Get(HeaderRemoteAddr); realIp != "" && allowedFrom(r) {
				r.RemoteAddr = realIp
			}
			handler.ServeHTTP(w, r)
		})
	}
}
