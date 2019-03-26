package main

import (
	"net/http"

	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/task"
)

func main() {
	srv := httpu.NewServer(&httpu.Config{
		Servers: []httpu.ServerConfig{
			{Addr: httpu.Addr(":9000")},
			{Addr: httpu.Addr(":9002")},
		},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello!"))
	}))

	task.NewRunner(srv).MustSigRun()
}
