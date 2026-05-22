package mtproto

import (
	"errors"
	"testing"
	"time"
)

func TestFloodWaitDuration(t *testing.T) {
	err := &RPCError{Code: 420, Message: "FLOOD_WAIT_3"}
	got, ok := FloodWaitDuration(err)
	if !ok || got != 3*time.Second {
		t.Fatalf("got %s %v", got, ok)
	}
	if IsFloodWait(errors.New("nope")) {
		t.Fatal("non flood wait matched")
	}
}

func TestBadServerSaltError(t *testing.T) {
	err := &BadServerSaltError{BadMessageID: 1, BadSeqNo: 2, Code: 48, NewSalt: 99}
	if err.Error() == "" {
		t.Fatal("empty error")
	}
}
