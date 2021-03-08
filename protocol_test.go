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

	seeds := []string {
		"121.41.85.45:33815",
	}
	err := StartTCPTurnServer(seeds)
	if err != nil {
		t.Error(err)
	}
}
