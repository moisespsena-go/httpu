package httpu

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/moisespsena-go/task"

	"github.com/moisespsena-go/logging"
)

type Listener struct {
	net.Listener

	Server      *http.Server
	KeepAlive   time.Duration
	Tls         *TlsConfig
	running     bool
	Log         logging.Logger
	stop        bool
	connections map[net.Conn]interface{}
	connWg      sync.WaitGroup
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

func (l *Listener) Shutdown(ctx context.Context) (err error) {
	l.mu.Lock()
	if l.stop {
		l.mu.Unlock()
		return
	}
	l.stop = true
	l.mu.Unlock()
	return l.shutdown(ctx)
}

func (l *Listener) shutdown(ctx context.Context) (err error) {
	l.mu.Lock()
	l.stop = true
	l.mu.Unlock()

	finished := make(chan struct{}, 1)
	go func() {
		l.connWg.Wait()
		finished <- struct{}{}
	}()

	defer l.Close()

	select {
	case <-ctx.Done():
		for c := range l.connections {
			c.Close()
		}
		return ctx.Err()
	case <-finished:
		return
	}

	return
}

func (l *Listener) ShutdownLog(ctx context.Context) (err error) {
	l.mu.Lock()
	if l.stop {
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()

	if err = l.shutdown(ctx); err != nil {
		if err != context.DeadlineExceeded {
			l.Log.Errorf("Listener shutdown failed: %v", err)
		}
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
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.stop || !l.running {
			return nil, io.EOF
		}
		con = &connection{con, func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			if _, ok := l.connections[con]; ok {
				delete(l.connections, con)
				l.connWg.Done()
			}
		}}
		if l.connections == nil {
			l.connections = map[net.Conn]interface{}{}
		}
		l.connWg.Add(1)
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
		return l.Server.ServeTLS(l, l.Tls.CertFile, l.Tls.KeyFile)
	}
	return l.Server.Serve(l)
}

func (l *Listener) Close() (err error) {
	l.mu.Lock()
	defer func() {
		l.running = false
		l.mu.Unlock()
	}()
	if !l.running {
		return
	}
	return l.Listener.Close()
}

type connection struct {
	net.Conn
	closer func()
}

func (con connection) Close() error {
	defer con.closer()
	return con.Conn.Close()
}
