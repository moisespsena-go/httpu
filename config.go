package httpu

import (
	"crypto/tls"
	"net/http"

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

type ServerConfig struct {
	Addr  Addr
	Tls   TlsConfig
	Http2 Http2Config
}

func (cfg *ServerConfig) CreateServer() (s *http.Server, err error) {
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
	Servers []ServerConfig
}
