package gop2p

import (
	"testing"
	"net"
//	"time"
	"fmt"
	"log"
)

/*
type netEvent struct {
	args [5]*interface{}
}

func (e *netEvent) OnRequest (args ...interface{}) error {
	e.args[
}

func (e *netEvent) OnNotice (args ...interface{}) error {
}

func (e *netEvent) OnOK (args ...interface{}) error {
}

func (e *netEvent) OnTurning (args ...interface{}) error {
}

func (e *netEvent) OnNotice2 (args ...interface{}) error {
}
*/

var event *Event_T

func init() {
	event = &Event_T{}
	event.Args[0] = [2]string{"on request", "hi hi hi"}
	event.Args[1] = "on Notice"
	event.Args[2] = [2]interface{}{"on OK", syncron}
	event.Args[3] = "on Turning"
	event.Args[4] = "on Notice 2"

	event.OnRequest = func(command int) error {
		for _, v := range event.Args[command].([2]string) {
			fmt.Println(v)
		}
		return nil
	}

	event.OnNotice = func(command int) error {
		fmt.Println(event.Args[command])
		return nil
	}

	event.OnOK = func(command int) error {
		fmt.Println(event.Args[command].([2]interface{})[0])
		event.Args[command].([2]interface{})[1].(func() error)()
		return nil
	}

	event.OnTurning = func(command int) error {
		fmt.Println(event.Args[command])
		return nil
	}

	event.OnNotice2 = func(command int) error {
		fmt.Println(event.Args[command])
		return nil
	}
}

func TestMain(t *testing.T) {

	seeds := []string {
		"121.41.85.45:34009",
	}

//	go sendDemo(t)

	err := StartTCPTurnServer(seeds, event, processLogic)
//	err := StartTCPTurnServer(seeds, eventsArrayFunc, processLogic)
	if err != nil {
		t.Error(err)
	}
}

func syncron() error {
	for peer, _ := range GetPeers() {
		err := Send(peer, 5, []byte("aaa"))
		if err != nil {
			return err
		}
	}

	return nil
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
