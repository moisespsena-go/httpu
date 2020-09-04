package httpu

import (
	"io"
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	// WroteHeader returns if has sent header to the client.
	WroteHeader() bool
	// Wrote returns if has sent bytes to the client.
	Wrote() bool
	// Status returns the HTTP status of the request, or 0 if one has not
	// yet been sent.
	Status() int
	// BytesWritten returns the total number of bytes sent to the client.
	BytesWritten() int
	// Unwrap returns the original proxied target.
	Unwrap() http.ResponseWriter
}

type TeeResponseWriter interface {
	http.ResponseWriter
	// Tee causes the response body to be written to the given io.Writer in
	// addition to proxying the writes through. Only one io.Writer can be
	// tee'd to at once: setting a second one will overwrite the first.
	// Writes will be sent to the proxy before being written to this
	// io.Writer. It is illegal for the tee'd writer to be modified
	// concurrently with writes.
	Tee(io.Writer)
}
