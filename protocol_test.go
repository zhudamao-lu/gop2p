package gop2p

import (
	"testing"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
//	"time"
	"fmt"
	"log"
)

var event *Event_T

type RPCCommandServer struct {}

func init() {
	event = &Event_T{}
	event.Args[0] = [2]string{"on request", "hi hi hi"}
	event.Args[1] = "on Notice"
//	event.Args[2] = [2]interface{}{"on OK", syncron}
	event.Args[2] = [2]interface{}{"on OK", nil}
	event.Args[3] = "on Turning"
	event.Args[4] = "on Notice 2"

	event.OnRequest = func(command int, innerArgs ...interface{}) error {
		for _, v := range event.Args[command].([2]string) {
			fmt.Println(v)
		}

		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此
		return nil
	}

	event.OnNotice = func(command int, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// event.SeedConn的赋值 在gop2p 内部完成

		return nil
	}

	event.OnOK = func(command int, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command].([2]interface{})[0])
		fmt.Println("peers:", GetPeers())
	//	event.Args[command].([2]interface{})[1].(func() error)()
		return nil
	}

	event.OnTurning = func(command int, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		return nil
	}

	event.OnNotice2 = func(command int, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// event.SeedConn的赋值 在gop2p 内部完成
		return nil
	}
}

func TestMain(t *testing.T) {
	seeds := []string {
		"121.41.85.45:39991",
	}

	go func() {
		err := startRPCCommandServer()
		if err != nil {
			t.Fatal(err)
		}
	}()

	err := StartTCPTurnServer(seeds, event, processLogic)
	if err != nil {
		t.Fatal(err)
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
		log.Println("api 0:", string(data))
	case 1:
		log.Println("api 1:", string(data))
	}

	return nil
}

func broadcast(api int, data []byte) error {
	return nil
}

func (c *RPCCommandServer)Api(args interface{}, reply *string) error {
	log.Println("conns:", GetPeers())
	Broadcast(5, []byte("testInfo"))

	return nil
}

func startRPCCommandServer() error {
	rpc.Register(&RPCCommandServer{})

	ln, err := net.Listen("tcp", ":1025")
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("rpc service is listening on port 1025")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err, conn)
			continue
		}

		jsonrpc.ServeConn(conn)
	}

	return nil
}
