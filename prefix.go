package httpu

import (
	"context"
	"net/http"
	"path"
	"strings"
)

func SetPrefix(ctx context.Context, pth string) context.Context {
	if pth != "" {
		if pth[0] != '/' {
			pth = "/" + pth
		}
		if pth[len(pth)-1] != '/' {
			pth += "/"
		}
	}
	return context.WithValue(ctx, CtxPrefix, pth)
}

func SetPrefixR(r *http.Request, pth string) *http.Request {
	return r.WithContext(SetPrefix(r.Context(), pth))
}

func PushPrefix(ctx context.Context, pth string) context.Context {
	if prefix := ctx.Value(CtxPrefix); prefix != nil {
		return SetPrefix(ctx, path.Join(prefix.(string), pth))
	}
	return SetPrefix(ctx, pth)
}

func PopPrefix(ctx context.Context, pth string) context.Context {
	if prefix := ctx.Value(CtxPrefix); prefix != nil {
		return SetPrefix(ctx, strings.TrimSuffix(prefix.(string), pth))
	}
	return ctx
}

func PushPrefixR(r *http.Request, pth string) *http.Request {
	return r.WithContext(PushPrefix(r.Context(), pth))
}

func Prefix(ctx context.Context) string {
	if prefix := ctx.Value(CtxPrefix); prefix != nil {
		return prefix.(string)
	}
	return "/"
}

func PrefixR(r *http.Request) string {
	return Prefix(r.Context())
}
