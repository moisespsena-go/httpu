package httpu

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/moisespsena-go/httpu/middlewares"
	"github.com/moisespsena-go/iputils"

	"github.com/moisespsena-go/task"

	"github.com/go-errors/errors"

	"github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/path-helpers"
	"github.com/op/go-logging"
)

var (
	pkg                 = path_helpers.GetCalledDir()
	log                 = defaultlogger.NewLogger(pkg)
	ErrNoListenersFound = errors.New("No listeners found")
)

type ContextKey int

const (
	DefaultUriPrefixHeader = "X-Uri-Prefix"

	CtxPrefix ContextKey = 1
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
	Config              *Config
	Handler             http.Handler
	handler             http.Handler
	listeners           Listeners
	log                 *logging.Logger
	listenerCallbacks   []func(lis *Listener)
	preSetup, postSetup []func(ta task.Appender) error

	tasks task.Slice
}

func NewServer(cfg *Config, handler http.Handler) *Server {
	s := &Server{Config: cfg, Handler: handler}
	s.SetLog(log)
	return s
}

func (s *Server) PreSetup(f ...func(ta task.Appender) error) {
	s.preSetup = append(s.preSetup, f...)
}

func (s *Server) GetPreSetup() []func(ta task.Appender) error {
	return s.preSetup
}

func (s *Server) PostSetup(f ...func(ta task.Appender) error) {
	s.postSetup = append(s.postSetup, f...)
}

func (s *Server) GetPostSetup() []func(ta task.Appender) error {
	return s.postSetup
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

func (s *Server) Prepare() (err error) {
	if s.Config.Prefix != "" && !strings.HasSuffix(s.Config.Prefix, "/") {
		s.Config.Prefix += "/"
	}
	if !s.Config.DisableStripRequestPrefix && s.Config.RequestPrefixHeader == "" {
		s.Config.RequestPrefixHeader = DefaultUriPrefixHeader
	}

	if !s.Config.DisableStripRequestPrefix || s.Config.Prefix != "" {
		s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var prefix = "/"
			if !s.Config.DisableStripRequestPrefix {
				if pfx := r.Header.Get(s.Config.RequestPrefixHeader); pfx != "" {
					prefix += pfx[1:]
				}
			}

			if s.Config.Prefix != "" {
				prefix += s.Config.Prefix[1:]
			}

			StripPrefix(w, r, s.Handler, prefix, !s.Config.DisableSlashPermanentRedirect)
		})
	} else {
		s.handler = s.Handler
	}

	if len(s.Config.ForwardedFor) > 0 {
		var proxiers []iputils.Container
		for _, ff := range s.Config.ForwardedFor {
			rg, err := ff.Range()
			if err != nil {
				return fmt.Errorf("ip range for forwarded_for %q failed: %s", string(ff), err.Error())
			}
			proxiers = append(proxiers, rg)
		}
		s.handler = middlewares.RemoteAddrHeaderMiddleware(func(r *http.Request) bool {
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				if ip := net.ParseIP(host); ip != nil {
					return iputils.Contains(proxiers, ip)
				}
			}
			return false
		})(s.handler)
	}
	return
}

func (s *Server) Setup(appender task.Appender) (err error) {
	if err = s.Prepare(); err != nil {
		return
	}

	for _, ps := range s.preSetup {
		if err = ps(appender); err != nil {
			return fmt.Errorf("server pre_setup failed: %v", err.Error())
		}
	}

	if len(s.listeners) == 0 {
		if err = s.InitListeners(); err != nil {
			return
		}
	}

	if len(s.listeners) == 0 {
		return ErrNoListenersFound
	}

	for _, ps := range s.postSetup {
		if err = ps(appender); err != nil {
			return fmt.Errorf("server post_setup failed: %v", err.Error())
		}
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
		var kl *KeepAliveListener
		if addr.IsUnix() {
			if _, err2 := os.Stat(addr.UnixPath()); err2 == nil {
				pth := addr.UnixPath()
				log.Info("Removing", pth)
				if err = os.Remove(pth); err != nil {
					return
				}
			}
		} else {
			kl = NewKeepAliveListener(nil)
			if cfg.KeepAliveInterval != nil {
				var dur time.Duration
				dur, err = cfg.KeepAliveInterval.Get()
				if err != nil {
					err = fmt.Errorf("get KeepAliveInterval failed: %v", err)
					return
				}
				if dur != 0 {
					kl.KeepAliveInterval = dur
				}
			}
			if cfg.KeepAliveIdleInterval != nil {
				var dur time.Duration
				dur, err = cfg.KeepAliveIdleInterval.Get()
				if err != nil {
					err = fmt.Errorf("get KeepAliveIdleInterval failed: %v", err)
					return
				}
				if dur != 0 {
					kl.KeepAliveIdleInterval = dur
				}
			}
			if cfg.KeepAliveCount != 0 {
				kl.KeepAliveCount = cfg.KeepAliveCount
			}
		}
		var l net.Listener
		if l, err = addr.CreateListener(); err != nil {
			return
		} else {
			log.Infof("listening on %s", l.Addr().String())

			if !addr.IsUnix() {
				kl.Listener = l
				l = kl
			}

			var srv *http.Server
			if srv, err = cfg.CreateServer(); err != nil {
				return
			}
			if s.handler == nil {
				srv.Handler = s.Handler
			} else {
				srv.Handler = s.handler
			}
			lis := &Listener{
				Server:   srv,
				Listener: l,
				Log:      defaultlogger.NewLogger(pkg + " L{" + string(cfg.Addr) + "}"),
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
