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
	if len(b) != 8 {
		return 0, errors.New("bytesToInt: bytes length must be 8")
	}

	return int(uint8(b[0])) << 56 | int(uint8(b[1])) << 48 | int(uint8(b[2])) << 40 | int(uint8(b[3])) << 32 | int(uint8(b[4])) << 24 | int(uint8(b[5])) << 16 | int(uint8(b[6])) << 8 | int(uint8(b[7])), nil
}

func intToBytes(i int) []byte {
	return []byte{uint8(i >> 56), uint8(i << 8 >> 56), uint8(i << 16 >> 56), uint8(i << 24 >> 56), uint8(i << 32 >> 56), uint8(i << 40 >> 56), uint8(i << 48 >> 56), uint8(i << 56 >> 56)}
}

func bytesToInt32(b []byte) (int32, error) {
	if len(b) != 4 {
		return 0, errors.New("bytesToInt4: bytes length must be 4")
	}

	return int32(uint8(b[0])) << 24 | int32(uint8(b[1])) << 16 | int32(uint8(b[2])) << 8 | int32(uint8(b[3])), nil
}

func int32ToBytes(i int32) []byte {
	return []byte{uint8(i >> 24), uint8(i << 8 >> 24), uint8(i << 16 >> 24), uint8(i << 24 >> 24)}
}

func bytesToInt64(b []byte) (int64, error) {
	i, err := bytesToInt(b)
	if err != nil {
		return 0, err
	}

	return int64(i), nil
}

func int64ToBytes(i int64) []byte {
	return intToBytes(int(i))
}
