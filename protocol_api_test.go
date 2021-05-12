package gop2p

import (
	"testing"
	"net/rpc/jsonrpc"
//	"fmt"
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

func TestApi(t *testing.T) {
	err := cmdHandle("Api", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}
