package httpu

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/moisespsena-go/task"

	"github.com/op/go-logging"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

type Listener struct {
	Server   *http.Server
	Listener net.Listener
	Tls      *TlsConfig
	running  bool
	Log      *logging.Logger
	stop     bool
}

func (l *Listener) Setup(appender task.Appender) error {
	return nil
}

func (l *Listener) Run() error {
	l.running = true
	defer func() {
		l.running = false
	}()
	return l.ListenAndServe()
}

func (l *Listener) Start(done func()) (stop task.Stoper, err error) {
	go func() {
		defer func() {
			done()
		}()
		if err := l.Run(); err != nil {
			if !l.stop {
				l.Log.Error(err)
			}
		} else {
			l.Log.Info("done")
		}
	}()
	return l, nil
}

func (l *Listener) Stop() {
	go func() {
		if l.stop {
			return
		}
		l.stop = true
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		if err := l.Server.Shutdown(ctx); err != nil {
			l.Log.Errorf("Server Shutdown error: %v", err)
		} else {
			l.Log.Info("Server gracefully stopped")
		}
	}()
}

func (l *Listener) IsRunning() bool {
	return l.running
}

func (l *Listener) ListenAndServe() error {
	if l.Tls != nil && l.Tls.Valid() {
		defer l.Listener.Close()
		return l.Server.ServeTLS(tcpKeepAliveListener{l.Listener.(*net.TCPListener)}, l.Tls.CertFile, l.Tls.KeyFile)
	}
	return l.Server.Serve(l.Listener)
}
