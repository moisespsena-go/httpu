package httpu

import (
	"crypto/tls"
	"net"
	"reflect"
	"time"
	"unsafe"
)

var (
	// DefaultKeepAliveIdleInterval specifies how long connection can be idle
	// before sending keepalive message.
	DefaultKeepAliveIdleInterval = 15 * time.Minute
	// DefaultKeepAliveCount specifies maximal number of keepalive messages
	// sent before marking connection as dead.
	DefaultKeepAliveCount = 8
	// DefaultKeepAliveInterval specifies how often retry sending keepalive
	// messages when no response is received.
	DefaultKeepAliveInterval = 5 * time.Second
)

type KeepAliveListener struct {
	net.Listener

	// DefaultKeepAliveCount specifies maximal number of keepalive messages
	// sent before marking connection as dead.
	KeepAliveCount int
	// DefaultKeepAliveIdleInterval specifies how long connection can be idle
	// before sending keepalive message.
	KeepAliveIdleInterval time.Duration
	// DefaultKeepAliveInterval specifies how often retry sending keepalive
	// messages when no response is received.
	KeepAliveInterval time.Duration
}

func NewKeepAliveListener(listener net.Listener) *KeepAliveListener {
	return &KeepAliveListener{
		Listener:              listener,
		KeepAliveCount:        DefaultKeepAliveCount,
		KeepAliveIdleInterval: DefaultKeepAliveIdleInterval,
		KeepAliveInterval:     DefaultKeepAliveInterval,
	}
}

func (ln KeepAliveListener) Accept() (net.Conn, error) {
	c, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}

	if tlsConn, ok := c.(*tls.Conn); ok {
		var t time.Time
		tlsConn.SetDeadline(t)
		tlsConn.SetWriteDeadline(t)
		v := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
		ptrToY := unsafe.Pointer(v.UnsafeAddr())
		realPtrToY := (*net.Conn)(ptrToY)

		if err = ln.keepAlive(*(realPtrToY)); err != nil {
			return nil, err
		}
	} else if err = ln.keepAlive(c); err != nil {
		return nil, err
	}
	return c, nil
}
