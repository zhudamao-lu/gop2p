package gop2p

import (
	"net"
	"testing"
)

func TestMain(t *testing.T) {
	err := Dial()
	if err != nil {
		t.Error(err)
	}
}

func Dial() error {
	conn, err := net.Dial("tcp", "121.41.85.45:1024")
	if err != nil {
		return err
	}

	head := []byte{0, 0, 0, 11, 1, 2, 3, 4}
	body := []byte("hello world")

	data := append(head, body...)
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}
