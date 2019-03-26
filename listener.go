package httpu

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/moisespsena-go/task"

	"github.com/op/go-logging"
)

type tcpKeepAliveListener struct {
	net.Listener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	c, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tc := c.(*connection).Conn.(*net.TCPConn)
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return c, nil
}

type Listener struct {
	net.Listener

	Server      *http.Server
	Tls         *TlsConfig
	running     bool
	Log         *logging.Logger
	stop        bool
	connections map[net.Conn]interface{}
	mu          sync.RWMutex
}

func (l *Listener) Connections() (cons []net.Conn) {
	if l.connections == nil {
		return
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	for con := range l.connections {
		cons = append(cons, con)
	}
	return
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
				l.Log.Error(err.Error())
			}
		} else {
			l.Log.Info("done")
		}
	}()
	return l, nil
}

func (l *Listener) Shutdown(ctx context.Context) error {
	l.stop = true
	return l.Server.Shutdown(ctx)
}

func (l *Listener) ShutdownLog(ctx context.Context) (err error) {
	if err = l.Shutdown(ctx); err != nil {
		if err == context.DeadlineExceeded {
			for con := range l.connections {
				con.Close()
			}
		}
		l.Log.Errorf("Listener shutdown failed: %v", err)
	} else {
		l.Log.Info("Listener gracefully stopped")
	}
	return
}

func (l *Listener) Stop() {
	if l.stop {
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	go l.ShutdownLog(ctx)
}

func (l *Listener) Accept() (con net.Conn, err error) {
	con, err = l.Listener.Accept()
	if err == nil {
		con = &connection{con, func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			delete(l.connections, con)
		}}
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.connections == nil {
			l.connections = map[net.Conn]interface{}{}
		}
		l.connections[con] = nil
	}
	return
}

func (l *Listener) IsRunning() bool {
	return l.running
}

func (l *Listener) ListenAndServe() error {
	if l.Tls != nil && l.Tls.Valid() {
		defer l.Listener.Close()
		return l.Server.ServeTLS(tcpKeepAliveListener{l}, l.Tls.CertFile, l.Tls.KeyFile)
	}
	return l.Server.Serve(l)
}

type connection struct {
	net.Conn
	closer func()
}

func (con connection) Close() error {
	defer con.closer()
	return con.Conn.Close()
}
