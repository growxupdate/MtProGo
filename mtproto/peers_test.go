package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestScanUserRefs(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorUser)
	b = binary.LittleEndian.AppendUint32(b, 1) // flags.0 access_hash
	b = binary.LittleEndian.AppendUint32(b, 0) // flags2
	b = binary.LittleEndian.AppendUint64(b, 12345)
	b = binary.LittleEndian.AppendUint64(b, 67890)

	users := scanUserRefs(b)
	if len(users) != 1 {
		t.Fatalf("users=%d", len(users))
	}
	if users[0].ID != 12345 || users[0].AccessHash != 67890 || users[0].Kind != "user" {
		t.Fatalf("unexpected user: %+v", users[0])
	}
}

func TestScanUpdatesState(t *testing.T) {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorUpdatesState)
	b = binary.LittleEndian.AppendUint32(b, 10)
	b = binary.LittleEndian.AppendUint32(b, 20)
	b = binary.LittleEndian.AppendUint32(b, 30)
	b = binary.LittleEndian.AppendUint32(b, 40)
	b = binary.LittleEndian.AppendUint32(b, 50)

	state := scanUpdatesState(b)
	if state == nil {
		t.Fatal("state is nil")
	}
	if state.PTS != 10 || state.QTS != 20 || state.Date != 30 || state.Seq != 40 || state.UnreadCount != 50 {
		t.Fatalf("unexpected state: %+v", state)
	}
}
