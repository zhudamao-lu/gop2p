package gop2p

import (
	"testing"
)

func TestMain(t *testing.T) {
	/*
	err := StartUDPTurnServer()
	if err != nil {
		t.Error(err)
	}
	*/

	err := StartTCPTurnServer()
	if err != nil {
		t.Error(err)
	}
}
