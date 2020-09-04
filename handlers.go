package httpu

import "net/http"

type FallbackHandlers []http.Handler

func (this *FallbackHandlers) Add(handler ...http.Handler) {
	*this = append(*this, handler...)
}

func (this FallbackHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wtd := ResponseWriterOf(w)
	for _, handler := range this {
		handler.ServeHTTP(wtd, r)
		if wtd.WroteHeader() {
			return
		}
	}
}

func Fallback(handlers ...http.Handler) http.Handler {
	return FallbackHandlers(handlers)
}
