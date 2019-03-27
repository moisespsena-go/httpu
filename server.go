package httpu

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/moisespsena-go/task"

	"github.com/go-errors/errors"

	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-path-helpers"
	"github.com/op/go-logging"
)

var (
	pkg                 = path_helpers.GetCalledDir()
	log                 = defaultlogger.NewLogger(pkg)
	ErrNoListenersFound = errors.New("No listeners found")
)

type Listeners []*Listener

func (l Listeners) Tasks() task.Slice {
	ts := make(task.Slice, len(l))
	for i, t := range l {
		ts[i] = t
	}
	return ts
}

type Server struct {
	Config            *Config
	Handler           http.Handler
	listeners         Listeners
	log               *logging.Logger
	listenerCallbacks []func(lis *Listener)

	tasks task.Slice
}

func NewServer(cfg *Config, handler http.Handler) *Server {
	s := &Server{Config: cfg, Handler: handler}
	s.SetLog(log)
	return s
}

func (s *Server) OnListener(f ...func(lis *Listener)) {
	s.listenerCallbacks = append(s.listenerCallbacks, f...)
}

func (s *Server) SetLog(log *logging.Logger) {
	s.log = log
}

func (s *Server) Listeners() []*Listener {
	return s.listeners
}

func (s *Server) Setup(appender task.Appender) (err error) {
	if len(s.listeners) == 0 {
		if err = s.InitListeners(); err != nil {
			return
		}
	}

	if len(s.listeners) == 0 {
		return ErrNoListenersFound
	}

	return
}

func (s *Server) Run() (err error) {
	return s.tasks.Run()
}

func (s *Server) Start(done func()) (stop task.Stoper, err error) {
	return task.Start(func(state *task.State) {
		done()
	}, s.tasks...)
}

func (s *Server) InitListeners() (err error) {
	var (
		listeners = make([]*Listener, len(s.Config.Listeners))
		tasks     = make(task.Slice, len(s.Config.Listeners))
	)

	defer func() {
		if err != nil {
			for _, l := range listeners {
				if l == nil {
					break
				}
				l.Close()
			}
		}
	}()

	log := s.log
	for i, cfg := range s.Config.Listeners {
		addr := cfg.Addr
		var keepAlive time.Duration
		if addr.IsUnix() {
			if _, err2 := os.Stat(addr.UnixPath()); err2 == nil {
				pth := addr.UnixPath()
				log.Info("Removing", pth)
				if err = os.Remove(pth); err != nil {
					return
				}
			}
		} else if cfg.KeepAlive != nil {
			keepAlive, err = cfg.KeepAlive.Get()
			if err != nil {
				err = fmt.Errorf("get tcp keep alive failed: %v", err)
				return
			}
		}
		var l net.Listener
		if l, err = addr.CreateListener(); err != nil {
			return
		} else {
			log.Infof("listening on %s", l.Addr().String())

			if !addr.IsUnix() {
				if keepAlive != 0 {
					l = tcpKeepAliveListener{l, keepAlive}
				} else if cfg.Tls.Valid() {
					l = tcpKeepAliveListener{l, 3 * time.Minute}
				}
			}

			var srv *http.Server
			if srv, err = cfg.CreateServer(); err != nil {
				return
			}
			srv.Handler = s.Handler
			lis := &Listener{
				Server:    srv,
				Listener:  l,
				KeepAlive: keepAlive,
				Log:       defaultlogger.NewLogger(pkg + " L{" + string(cfg.Addr) + "}"),
			}
			if cfg.Tls.Valid() {
				lis.Tls = &TlsConfig{cfg.Tls.CertFile, cfg.Tls.KeyFile, cfg.Tls.NPNDisabled}
			}
			for _, cb := range s.listenerCallbacks {
				cb(lis)
			}
			listeners[i] = lis
			tasks[i] = lis
		}
	}
	s.listeners = listeners
	s.tasks = tasks
	return
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	for _, l := range s.listeners[1:] {
		go l.ShutdownLog(ctx)
	}

	return s.listeners[0].ShutdownLog(ctx)
}
