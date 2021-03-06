package httpu

import (
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type responseWriter struct {
	http.ResponseWriter
	wroteHeader, wrote bool
	bytesWritten       int
	status             int
	tee                []io.Writer
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (this *responseWriter) WriteHeader(s int) {
	if !this.wroteHeader {
		this.wroteHeader = true
		this.ResponseWriter.WriteHeader(s)
		this.status = s
	}
}

func (this *responseWriter) Write(p []byte) (n int, err error) {
	if !this.wrote {
		this.WriteHeader(200)
		this.wrote = true
	}
	n, err = this.ResponseWriter.Write(p)
	this.bytesWritten += n

	for i, tee := range this.tee {
		_, err2 := tee.Write(p[:n])
		// Prefer errors generated by the proxied writer.
		if err == nil {
			err = errors.Wrapf(err2, "tee[%d]", i)
		}
	}
	return
}

func (this *responseWriter) WroteHeader() bool {
	return this.wroteHeader
}

func (this *responseWriter) Wrote() bool {
	return this.wrote
}

func (this *responseWriter) BytesWritten() int {
	return this.bytesWritten
}

func (this *responseWriter) Status() int {
	return this.status
}

func (this *responseWriter) Tee(w io.Writer) {
	this.tee = append(this.tee, w)
}

func (this *responseWriter) Unwrap() http.ResponseWriter {
	return this.ResponseWriter
}

type teeResponseWriter struct {
	http.ResponseWriter
	tee []io.Writer
}

func NewTeeResponseWriter(w http.ResponseWriter, tee ...io.Writer) *teeResponseWriter {
	return &teeResponseWriter{ResponseWriter: w, tee: tee}
}

func (this *teeResponseWriter) Tee(w io.Writer) {
	this.tee = append(this.tee, w)
}

func (this *teeResponseWriter) Write(p []byte) (n int, err error) {
	n, err = this.ResponseWriter.Write(p)
	for i, tee := range this.tee {
		_, err2 := tee.Write(p[:n])
		// Prefer errors generated by the proxied writer.
		if err == nil {
			err = errors.Wrapf(err2, "tee[%d]", i)
		}
	}
	return
}

func ResponseWriterOf(w http.ResponseWriter) (wd ResponseWriter) {
	if wd, ok := w.(ResponseWriter); ok {
		return wd
	}
	return NewResponseWriter(w)
}

func TeeResponseWriterOf(w http.ResponseWriter) (wd TeeResponseWriter) {
	if tw, ok := w.(TeeResponseWriter); ok {
		return tw
	}
	return NewTeeResponseWriter(w)
}
