/*
	+build !windows
*/
package gop2p

import (
	"syscall"
	"errors"

	"golang.org/x/sys/unix"
)

func controlSockReusePortUnix(network, address string, c syscall.RawConn) (err error) {
	return c.Control(func(fd uintptr) {
		// SO_REUSEADDR
		unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			panic(err)
		}

		// SO_REUSEPORT
		unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			panic(err)
		}
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
