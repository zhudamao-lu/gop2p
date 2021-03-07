package gop2p

import (
	"testing"
	"io"
	"net"
	"log"
)

const (
	PACKET_IDENTIFY = "--TCPHEADX--" // 包开始标识符
	PACKET_HEAD_LEN = 0x08 // 包头固定长度 4 + 4 = 8
	PACKET_COMMAND_LEN = 0x04 // 包头中命令固定长度 4
	PACKET_BODY_DATA_LEN = 0x04 // 包头中记录数据的固定长度 4
)

func TestMain(t *testing.T) {
	rAddr, err := net.ResolveTCPAddr("tcp4", "121.41.85.45:44807")
	if err != nil {
		t.Error(err)
	}
	log.Println("rAddr:", rAddr)

	conn, err := net.DialTCP("tcp", nil, rAddr)
//	conn, err := net.Dial("tcp", "116.231.30.130:63425")
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	body := []byte("hello brother hello server abcdefghijklmnopqrstuvwxyz1234567890")
	data := []byte(PACKET_IDENTIFY)
	data = append(data, intToBytes(0xffffffff)...)
	data = append(data, intToBytes(len(body))...)
	data = append(data, body...)
	log.Println(data)

	for i := 0; i < 100; i++ {
		n, err := conn.Write(data)
		if err != nil {
			t.Log(err)
		}
		log.Println(n)
	}

	for {
		data := make([]byte, 0, 4096)
		for {
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				t.Log(err)
				if err == io.EOF {
					return
				}
				break
			}
			log.Println(n)
			data = append(data, buffer[:n]...)
		}
		t.Log(data)
	}
}
