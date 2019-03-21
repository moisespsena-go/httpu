package httpu

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type Addr string

func (a Addr) IsUnix() bool {
	return strings.HasPrefix(string(a), "unix:")
}
func (a Addr) UnixPath() string {
	return strings.TrimPrefix(string(a), "unix:")
}

func (a Addr) Network() (net string, addr string) {
	parts := strings.SplitN(string(a), ":", 2)
	switch parts[0] {
	case "tcp", "tcp4", "tcp6", "unix":
		return parts[0], parts[1]
	default:
		return "tcp", string(a)
	}
}

func (a Addr) CreateListener() (net.Listener, error) {
	network, addr := a.Network()
	if network == "unix" {
		if _, err := os.Stat(addr); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else if err = syscall.Unlink(addr); err != nil {
			return nil, err
		}
	}
	return net.Listen(network, strings.TrimPrefix(string(a), network+":"))
}

func (a Addr) Port() int {
	_, addr := a.Network()
	parts := strings.Split(addr, ":")
	if port, err := strconv.Atoi(parts[1]); err != nil {
		panic(fmt.Errorf("Invalid addr %q: %v", addr, err))
	} else {
		return port
	}
}
