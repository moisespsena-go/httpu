// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

// +build !windows

package httpu

import (
	"net"

	"github.com/felixge/tcpkeepalive"
)

func (l KeepAliveListener) keepAlive(conn net.Conn) error {
	return tcpkeepalive.SetKeepAlive(conn, l.KeepAliveIdleInterval, l.KeepAliveCount, l.KeepAliveInterval)
}
