package gop2p

import (
	"testing"
	"net"
//	"time"
	"log"
)

var tt *testing.T
func TestMain(t *testing.T) {
	tt = t

	seeds := []string {
		"121.41.85.45:45647",
	}

//	go sendDemo(t)

	EventsArrayFunc[ACTION_CONNECTION_REQUEST] = nil /* func(args ...interface{}) error {
		log.Println("on request")
		return nil
	} */

	EventsArrayFunc[ACTION_CONNECTION_NOTICE] = func(args ...interface{}) error {
		log.Println("on notice")
		return nil
	}

	EventsArrayFunc[ACTION_CONNECTION_TURN_OK] = func(args ...interface{}) error {
		log.Println("on OK")
		return nil
	}

	EventsArrayFunc[ACTION_CONNECTION_TURNING] = func(args ...interface{}) error {
		log.Println("on turning")
		return nil
	}

	EventsArrayFunc[ACTION_CONNECTION_NOTICE2] = func(args ...interface{}) error {
		log.Println("on notice2")
		for k, v := range GetPeers() {
			log.Println("peer:", k.RemoteAddr(), v, "iiiiiiiiii")
		}
		return nil
	}

	err := StartTCPTurnServer(seeds, processLogic)
//	err := StartTCPTurnServer(seeds, eventsArrayFunc, processLogic)
	if err != nil {
		t.Error(err)
	}
}

/*
func syncron() error {
	for peer, _ := range GetPeers() {
		err := Send(peer, 5, []byte("aaa"))
		if err != nil {
			return err
		}
	}

	return nil
}
*/

func processLogic(api int, data []byte, conn *net.TCPConn) error {
	switch api {
	case 0:
		log.Println("case 0:", string(data))
	case 1:
		log.Println("case 1:", string(data))
	}

	return nil
}

	/*
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
	*/
