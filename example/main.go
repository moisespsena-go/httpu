package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/task"
)

func main() {
	srv := httpu.NewServer(&httpu.Config{
		Servers: []httpu.ServerConfig{
			{Addr: httpu.Addr(":9000")},
			{Addr: httpu.Addr(":9002"), Tls: httpu.TlsConfig{CertFile: "server.crt", KeyFile: "server.key"}},
		},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		println("request start")
		defer println("request done")
		<-time.After(10 * time.Second)
		w.Write([]byte("hello!"))
	}))

	go func() {
		for {
			<-time.After(2 * time.Second)
			for _, l := range srv.Listeners() {
				if count := len(l.Connections()); count > 0 {
					fmt.Println(l.Listener.Addr().String(), ":", count)
				}
			}
		}
	}()

	go func() {
		<-time.After(20 * time.Second)
		println("closing")
		ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
		log.Println(srv.Shutdown(ctx))
		println("closed")
	}()

	task.NewRunner(srv).MustSigRun()
}
