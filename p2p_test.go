package gop2p

import (
	"testing"
)

func TestMain(t *testing.T) {
	err := Start(":1024")
	if err != nil {
		t.Error(err)
	}
}
