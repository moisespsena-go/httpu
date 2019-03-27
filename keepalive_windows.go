package httpu

import (
	"fmt"
	"net"
)

func (l KeepAliveListener) keepAlive(conn net.Conn) error {
	c, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("Bad connection type: %T", c)
	}

	if err := c.SetKeepAlive(true); err != nil {
		return err
	}

	return nil
}
