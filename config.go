package httpu

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/http2"
)

type Http2Config struct {
	Disabled bool
	Config   *http2.Server
}

type TlsConfig struct {
	CertFile    string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile     string `mapstructure:"key_file" yaml:"key_file"`
	NPNDisabled bool
}

func (tls *TlsConfig) Valid() bool {
	return tls.CertFile != "" && tls.KeyFile != ""
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
	Addr      Addr
	Tls       TlsConfig
	Http2     Http2Config
	KeepAlive *KeepAliveConfig
}

func (cfg *ListenerConfig) CreateServer() (s *http.Server, err error) {
	s = &http.Server{}
	if !cfg.Http2.Disabled {
		if cfg.Tls.NPNDisabled {
			s.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
		}
		if err = http2.ConfigureServer(s, cfg.Http2.Config); err != nil {
			return nil, err
		}
	}
	return
}

type Config struct {
	Listeners []ListenerConfig
}
