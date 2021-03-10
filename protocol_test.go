package gop2p

import (
	"testing"
	"net"
	"time"
	"log"
)

var tt *testing.T
func TestMain(t *testing.T) {
	tt = t

	seeds := []string {
	//	"121.41.85.45:39279",
	}

	go sendDemo(t)

	err := StartTCPTurnServer(seeds, processLogic)
	if err != nil {
		t.Error(err)
	}
}

func processLogic(api int, data []byte, conn *net.TCPConn) error {
	switch api {
	case 0:
		log.Println("case 0:", string(data))
	case 1:
		log.Println("case 1:", string(data))
	}

	return nil
}

func sendDemo(t *testing.T) {
	time.Sleep(time.Second * 5)

	t.Log("sendDemo...")
	for k, _ := range peers {
		t.Log("conn:", k.LocalAddr(), k.RemoteAddr())
		body := intToBytes(0)
		body = append(body, []byte("hi hi hi")...)
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_LOGIC)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err := k.Write(sendData)
		if err != nil {
			t.Log(err)
			break
		}
		log.Println("n:", n)

		body = intToBytes(1)
		body = append(body, []byte("hello hello hello")...)
		sendData = []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_LOGIC)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err = k.Write(sendData)
		if err != nil {
			t.Log(err)
			break
		}
		log.Println("n:", n)
	}
}
