package gop2p

import (
	"testing"
)

func TestMain(t *testing.T) {
	err := StartTurnServer()
	if err != nil {
		t.Error(err)
	}
}
