package gop2p

import (
	"testing"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"encoding/hex"
	"time"
	"fmt"
	"log"
)

// var Event *Event_T

type RPCCommandServer struct {}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	Event = &Event_T{}
	Event.Args[0] = [2]string{"on request", "hi hi hi"}
	Event.Args[1] = "on Notice"
	Event.Args[2] = [2]interface{}{"on OK", nil}
	Event.Args[3] = "on Turning"
	Event.Args[4] = "on Notice 2"

	Event.OnRequest = func(command uint8, innerArgs ...interface{}) error {
		for _, v := range Event.Args[command].([2]string) {
			fmt.Println(v)
		}

		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此
		return nil
	}

	Event.OnResponse = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(Event.Args[command])
		fmt.Println(innerArgs[0].(*net.TCPConn))
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此
	//	Event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
	//	Event.SeedConn的赋值 在gop2p 内部完成

		fmt.Println("Event.OnResponse")

		return nil
	}

	Event.OnNotice = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(Event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// Event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// Event.SeedConn的赋值 在gop2p 内部完成

		return nil
	}

	Event.OnOK = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(Event.Args[command].([2]interface{})[0])
		fmt.Println("peers:", GetPeers())
	//	Event.Args[command].([2]interface{})[1].(func() error)()
		return nil
	}

	Event.OnTurning = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(Event.Args[command])
		return nil
	}

	Event.OnNotice2 = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(Event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// Event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// Event.SeedConn的赋值 在gop2p 内部完成
		return nil
	}

	Event.OnDisconnect = func(peer *net.TCPConn) {
		fmt.Println("OnDisconnect")
		fmt.Println(peer.RemoteAddr().String())
	}

	ProcessLogic = processLogic
}

func TestMain(t *testing.T) {
	seeds := []string {
	//	"47.117.44.142:42277", // mtcoin1
        //	"139.196.181.139:42555", // mtcoin2
		"47.98.204.151:11111", // mtcoin3
	}

	go func() {
		err := startRPCCommandServer()
		if err != nil {
			t.Fatal(err)
		}
	}()

	err := StartTCPTurnServer(seeds)
	if err != nil {
		t.Fatal(err)
	}
}

func processLogic(head, body []byte, conn *net.TCPConn) {
	api, err := GetApiFromBody(body)
	if err != nil {
		fmt.Println(err)
		return
	}

	data := GetDataFromBody(body)

	switch api {
	case 0:
		log.Println("api 0:", string(data))
	case 1:
		log.Println("api 1:", string(data))
	}

	Forward(append(head, body...), conn) // forward
}

func (c *RPCCommandServer)Api(args interface{}, reply *string) error {
	log.Println("conns:", GetPeers())
	for {
		time.Sleep(time.Duration(200) * time.Millisecond)
		go Broadcast(6, make([]byte, 0))
	}

	return nil
}

func (c *RPCCommandServer)GetHashNonces(args interface{}, reply *string) error {
	hashNonce := hashNonceFirst
	if hashNonce == nil {
		return nil
	}
	*reply += fmt.Sprintln("hashNonce hash:", hex.EncodeToString(hashNonce.hash))
	*reply += fmt.Sprintln("hashNonce nonce:", hex.EncodeToString(hashNonce.nonce))
	*reply += fmt.Sprintln("hashNonce time:", hashNonce.timestamp)
	*reply += "-------------------------------\n"
	hashNonce = hashNonce.next

	for hashNonce != hashNonceFirst {
		/*
		fmt.Println(hashNonce)
		fmt.Println(hashNonceFirst)
		*/
		*reply += fmt.Sprintln("hashNonce hash:", hex.EncodeToString(hashNonce.hash))
		*reply += fmt.Sprintln("hashNonce nonce:", hex.EncodeToString(hashNonce.nonce))
		*reply += fmt.Sprintln("hashNonce time:", hashNonce.timestamp)
		*reply += "-------------------------------\n"
		hashNonce = hashNonce.next
	}

	fmt.Println(*reply)

	return nil
}

func startRPCCommandServer() error {
	rpc.Register(&RPCCommandServer{})

	port := ":1025"
//	port := ":1026"
//	port := ":1027"

	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("rpc service is listening on port " + port)

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
