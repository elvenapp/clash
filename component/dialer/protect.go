//go:build foss

package dialer

import "syscall"

func protectSocketFunc(protectSocket func(fd uintptr) error) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var innerErr error

		err := c.Control(func(fd uintptr) {
			innerErr = protectSocket(fd)
		})
		if err != nil {
			return err
		} else if innerErr != nil {
			return innerErr
		}

		return nil
	}
}
