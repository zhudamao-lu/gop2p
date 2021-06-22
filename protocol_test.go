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

var event *Event_T

type RPCCommandServer struct {}

func init() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	event = &Event_T{}
	event.Args[0] = [2]string{"on request", "hi hi hi"}
	event.Args[1] = "on Notice"
//	event.Args[2] = [2]interface{}{"on OK", syncron}
	event.Args[2] = [2]interface{}{"on OK", nil}
	event.Args[3] = "on Turning"
	event.Args[4] = "on Notice 2"

	event.OnRequest = func(command uint8, innerArgs ...interface{}) error {
		for _, v := range event.Args[command].([2]string) {
			fmt.Println(v)
		}

		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此
		return nil
	}

	event.OnNotice = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// event.SeedConn的赋值 在gop2p 内部完成

		return nil
	}

	event.OnOK = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command].([2]interface{})[0])
		fmt.Println("peers:", GetPeers())
	//	event.Args[command].([2]interface{})[1].(func() error)()
		return nil
	}

	event.OnTurning = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		return nil
	}

	event.OnNotice2 = func(command uint8, innerArgs ...interface{}) error {
		fmt.Println(event.Args[command])
		AddPeer(innerArgs[0].(*net.TCPConn)) // 如果穿透服也是一个节点，则可以执行此

		// event.SeedConn.Close() // 如果穿透服不是一个节点，则关闭
		// event.SeedConn的赋值 在gop2p 内部完成
		return nil
	}
}

func TestMain(t *testing.T) {
	seeds := []string {
		"47.117.44.142:42277", // mtcoin1
        //	"139.196.181.139:42555", // mtcoin2
	//	"47.98.204.151:34553", // mtcoin3
	}

	go func() {
		err := startRPCCommandServer()
		if err != nil {
			t.Fatal(err)
		}
	}()
	go func (){
		for {
			for _,v := range peers{
				tmpV := <- v.C
				log.Println("read chan value: ", tmpV)
			}
			time.Sleep(1 * time.Second)
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

func processLogic(head, body []byte, conn *net.TCPConn) error {
	api, err := GetApiFromBody(body)
	if err != nil {
		return err
	}

	data := GetDataFromBody(body)

	switch api {
	case 0:
		log.Println("api 0:", string(data))
	case 1:
		log.Println("api 1:", string(data))
	}

	Forward(append(head, body...)) // forward

	return nil
}

func (c *RPCCommandServer)Api(args interface{}, reply *string) error {
	log.Println("conns:", GetPeers())
	// Broadcast(5, []byte("testInfo"))
	Broadcast(5, make([]byte,1024000000))

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
		fmt.Println(hashNonce)
		fmt.Println(hashNonceFirst)
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
