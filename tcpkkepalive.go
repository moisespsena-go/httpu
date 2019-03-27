package httpu

import (
	"net"
	"time"
)

type tcpKeepAliveListener struct {
	net.Listener
	duration time.Duration
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	c, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tc := c.(*connection).Conn.(*net.TCPConn)
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(ln.duration)
	return c, nil
}
