//go:build foss && linux

package dialer

import (
	"net"
	"syscall"
)

func bindMarkToDialer(mark int, dialer *net.Dialer, _ string, _ net.IP) {
	dialer.Control = bindMarkToControl(mark, dialer.Control)
}

func bindMarkToListenConfig(mark int, lc *net.ListenConfig, _, address string) {
	lc.Control = bindMarkToControl(mark, lc.Control)
}

func bindMarkToControl(mark int, chain controlFn) controlFn {
	return func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		ipStr, _, err := net.SplitHostPort(address)
		if err == nil {
			ip := net.ParseIP(ipStr)
			if ip != nil && !ip.IsGlobalUnicast() {
				return
			}
		}

		var innerErr error
		err = c.Control(func(fd uintptr) {
			innerErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_MARK, mark)
		})
		if innerErr != nil {
			err = innerErr
		}
		return
	}
}