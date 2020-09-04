package httpu

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"oliva.pw/tlsgen"

	"golang.org/x/net/http2"
)

type Http2Config struct {
	Disabled bool
	Config   *http2.Server
}

type TlsConfig struct {
	Generate    *tlsgen.Config `mapstructure:"generate" yaml:"generate"`
	CertFile    string         `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile     string         `mapstructure:"key_file" yaml:"key_file"`
	NPNDisabled bool
}

func (cfg *TlsConfig) Valid() bool {
	return cfg.CertFile != "" && cfg.KeyFile != ""
}

// KeepAliveConfig TCP keep alive duration.
// A duration string is a possibly signed value (seconds)
// or signed sequence of decimal numbers, each with optional
// fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
type KeepAliveConfig struct {
	Duration time.Duration
	Value    string `mapstructure:"str" yaml:"str"`
}

func (ka KeepAliveConfig) Get() (dur time.Duration, err error) {
	if ka.Duration == 0 {
		if ka.Value != "" {
			if secs, err := strconv.Atoi(ka.Value); err == nil {
				dur = time.Duration(secs) * time.Second
				return dur, nil
			}
			return time.ParseDuration(ka.Value)
		}
	}
	return
}

type ListenerConfig struct {
	Addr  Addr
	Tls   *TlsConfig
	Http2 Http2Config

	// DefaultKeepAliveCount specifies maximal number of keepalive messages
	// sent before marking connection as dead.
	KeepAliveCount int
	// DefaultKeepAliveIdleInterval specifies how long connection can be idle
	// before sending keepalive message.
	KeepAliveIdleInterval *KeepAliveConfig
	// DefaultKeepAliveInterval specifies how often retry sending keepalive
	// messages when no response is received.
	KeepAliveInterval *KeepAliveConfig

	Timeouts TimeoutsConfig `mapstructure:"timeouts" yaml:"timeouts"`
}

func (cfg *ListenerConfig) CreateServer() (s *http.Server, err error) {
	cfg.Timeouts.init()
	s = &http.Server{
		IdleTimeout:       cfg.Timeouts.IdleTimeout,
		ReadTimeout:       cfg.Timeouts.ReadTimeout,
		ReadHeaderTimeout: cfg.Timeouts.ReadHeaderTimeout,
		WriteTimeout:      cfg.Timeouts.WriteTimeout,
		MaxHeaderBytes:    cfg.Timeouts.MaxHeaderBytes,
	}
	if !cfg.Http2.Disabled && cfg.Tls != nil {
		if cfg.Tls.NPNDisabled {
			s.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
		}
		if err = http2.ConfigureServer(s, cfg.Http2.Config); err != nil {
			return nil, err
		}
	}
	return
}

type TimeoutsConfig struct {
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`

	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body. If ReadHeaderTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout" yaml:"read_header_timeout"`

	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	WriteTimeout time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	IdleTimeout time.Duration `mapstructure:"idle_timeout" yaml:"idle_timeout"`

	// MaxHeaderBytes controls the maximum number of bytes the
	// server will read parsing the request header's keys and
	// values, including the request line. It does not limit the
	// size of the request body.
	// If zero, DefaultMaxHeaderBytes is used.
	MaxHeaderBytes int `mapstructure:"max_header_bytes" yaml:"max_header_bytes"`
}

func (this *TimeoutsConfig) init() {
	set := func(v *time.Duration, defaul time.Duration) {
		if *v == 0 {
			*v = defaul
		} else if *v == -1 {
			*v = 0
		}
	}
	set(&this.IdleTimeout, 2*time.Minute)
	set(&this.ReadHeaderTimeout, 3*time.Second)
	set(&this.ReadTimeout, 10*time.Second)
}

type Config struct {
	Listeners []ListenerConfig

	Prefix                        string
	RequestPrefixHeader           string `mapstructure:"request_prefix_header" yaml:"request_prefix_header"`
	DisableStripRequestPrefix     bool   `mapstructure:"disable_strip_request_prefix" yaml:"disable_strip_request_prefix"`
	DisableSlashPermanentRedirect bool   `mapstructure:"disable_slash_permanent_redirect" yaml:"disable_slash_permanent_redirect"`
	MaxPostSize                   int64  `mapstructure:"max_post_size" yaml:"max_post_size"`
	UnlimitedPostSize             bool   `mapstructure:"unlimited_request_size" yaml:"unlimited_post_size"`
	NotFoundDisabled              bool   `mapstructure:"not_found_disabled" yaml:"not_found_disabled"`
}
