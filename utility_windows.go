//	+build windows

package gop2p

import (
	"syscall"
	"errors"

	"golang.org/x/sys/windows"
)

func controlSockReusePortWindows(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	})
}

func bytesToInt(b []byte) (int, error) {
	if len(b) != 4 {
		return 0, errors.New("bytes length must be 4")
	}

	return int(uint8(b[0])) << 24 | int(uint8(b[1])) << 16 | int(uint8(b[2])) << 8 | int(uint8(b[3])), nil
}

func intToBytes(i int) ([]byte) {
	return []byte{uint8(i >> 24), uint8(i << 8 >> 24), uint8(i << 16 >> 24), uint8(i << 24 >> 24)}
}
