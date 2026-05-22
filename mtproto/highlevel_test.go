package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestMakeMessagesSendMessageWithOptions(t *testing.T) {
	q, err := makeMessagesSendMessageQueryWithOptions(InputPeerUser(10, 20), "hi", SendOptions{ReplyToMessageID: 7, Silent: true, NoWebpage: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(q[:4]); got != constructorMessagesSendMessage {
		t.Fatalf("constructor = 0x%x", got)
	}
	flags := binary.LittleEndian.Uint32(q[4:8])
	want := uint32(1<<0 | 1<<1 | 1<<5)
	if flags != want {
		t.Fatalf("flags = %b, want %b", flags, want)
	}
}

func TestMakeMessagesForwardMessagesQuery(t *testing.T) {
	q, err := makeMessagesForwardMessagesQuery(InputPeerChat(1), InputPeerChat(2), []int32{3, 4}, ForwardOptions{Silent: true, DropAuthor: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(q[:4]); got != constructorMessagesForward {
		t.Fatalf("constructor = 0x%x", got)
	}
	flags := binary.LittleEndian.Uint32(q[4:8])
	want := uint32(1<<5 | 1<<11)
	if flags != want {
		t.Fatalf("flags = %b, want %b", flags, want)
	}
}

func TestScanUsers(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorVector)
	b.putInt(1)
	b.putInt(constructorUser)
	b.putInt(1<<0 | 1<<1 | 1<<3 | 1<<10 | 1<<14)
	b.putInt(0)
	b.putLong(123)
	b.putLong(456)
	b.putString("Test")
	b.putString("testbot")
	users := ScanUsers(b.bytes())
	if len(users) != 1 {
		t.Fatalf("users len = %d", len(users))
	}
	if users[0].ID != 123 || users[0].AccessHash != 456 || users[0].Username != "testbot" || !users[0].Bot || !users[0].Self {
		t.Fatalf("bad user: %+v", users[0])
	}
}
