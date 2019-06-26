package httpu

import (
	"net/http"
	"sort"
	"strings"
)

type PrefixHandlers []struct {
	prefix  string
	handler http.Handler
}

func (this PrefixHandlers) Get(uri string) http.Handler {
	if this == nil {
		return nil
	}
	for _, el := range this {
		if strings.HasPrefix(uri, el.prefix) {
			return el.handler
		}
	}
	return nil
}

func (this *PrefixHandlers) Set(prefix string, handler http.Handler) {
	for _, el := range *this {
		if el.prefix == prefix {
			el.handler = handler
			return
		}
	}
	(*this) = append(*this, struct {
		prefix  string
		handler http.Handler
	}{prefix: prefix, handler: handler})
	sort.Slice(*this, func(i, j int) bool {
		return (*this)[i].prefix > (*this)[j].prefix
	})
}

func (this *PrefixHandlers) With(prefix string, handler http.Handler) {
	this.Set(prefix, PrefixHandler(prefix, handler))
}

func (this PrefixHandlers) ServeHttpHandler(w http.ResponseWriter, r *http.Request, defaul http.Handler) {
	if handler := this.Get(r.URL.Path); handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		defaul.ServeHTTP(w, r)
	}
}

func (this PrefixHandlers) ServeHTTP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		this.ServeHttpHandler(w, r, http.NotFoundHandler())
	})
}
