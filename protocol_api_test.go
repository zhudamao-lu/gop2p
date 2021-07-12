package gop2p

import (
	"testing"
	"net/rpc/jsonrpc"
	"fmt"
	"log"
)

func cmdHandle(methodName string, args interface{}, reply interface{}) error {
	client, err := jsonrpc.Dial("tcp", ":1025")
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.Call("RPCCommandServer." + methodName, args, &reply)
	if err != nil {
		return err
	}

	return nil
}

// func TestApi(t *testing.T) {
// 	var reply string
// //	err := cmdHandle("Api", nil, nil)
// 	err := cmdHandle("GetHashNonces", nil, &reply)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println(reply)
// }

func TestApi(t *testing.T) {
	var reply string
//	err := cmdHandle("Api", nil, nil)
	err := cmdHandle("Api", nil, &reply)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(reply)
}