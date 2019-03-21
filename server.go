package httpu

import (
	"net/http"
	"syscall"

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
	task.Tasks
	Config            *Config
	Handler           http.Handler
	listeners         Listeners
	log               *logging.Logger
	listenerCallbacks []func(lis *Listener)
}

func NewServer(cfg *Config, handler http.Handler) *Server {
	s := &Server{Config: cfg, Handler: handler}
	s.PreRun(func(ta task.Appender) (err error) {
		if len(s.listeners) == 0 {
			if err = s.InitListeners(); err != nil {
				return
			}
		}

		if len(s.listeners) == 0 {
			return ErrNoListenersFound
		}
		return ta.AddTask(s.listeners.Tasks()...)
	})
	s.SetLog(log)
	return s
}

func (s *Server) OnListener(f ...func(lis *Listener)) {
	s.listenerCallbacks = append(s.listenerCallbacks, f...)
}

func (s *Server) SetLog(log *logging.Logger) {
	s.log = log
	s.Tasks.SetLog(log)
}

func (s *Server) Listeners() []*Listener {
	return s.listeners
}

func (s *Server) Start(done func()) (stop task.Stoper, err error) {
	var ostop task.Stoper
	ostop, err = s.Tasks.Start(func() {
		s.log.Info("done.")
		done()
	})
	if err == nil {
		stop = task.NewStoper(func() {
			defer ostop.Stop()
			s.log.Info("stop required")
		}, ostop.IsRunning)
	}
	return
}

func (s *Server) InitListeners() (err error) {
	var listeners = make([]*Listener, len(s.Config.Servers))
	log := s.log
	for i, srvCfg := range s.Config.Servers {
		addr := srvCfg.Addr
		if addr.IsUnix() {
			defer func(pth string) func() {
				return func() {
					log.Info("Removing", pth)
					if err := syscall.Unlink(pth); err != nil {
						log.Errorf("Removing %q failed: %s", pth, err)
					}
				}
			}(addr.UnixPath())
		}
		log.Infof("Creating listener of %q", addr)
		if l, err := addr.CreateListener(); err != nil {
			return err
		} else {
			var srv *http.Server
			if srv, err = srvCfg.CreateServer(); err != nil {
				return err
			}
			srv.Handler = s.Handler
			lis := &Listener{Server: srv, Listener: l, Log: defaultlogger.NewLogger(pkg + " L{" + string(srvCfg.Addr) + "}")}
			if srvCfg.Tls.Valid() {
				lis.Tls = &TlsConfig{srvCfg.Tls.CertFile, srvCfg.Tls.KeyFile, srvCfg.Tls.NPNDisabled}
			}
			for _, cb := range s.listenerCallbacks {
				cb(lis)
			}
			listeners[i] = lis
		}
	}
	s.listeners = listeners
	return nil
}
