package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestParseDifferenceEmpty(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorDifferenceEmpty)
	b.putInt(10)
	b.putInt(20)
	diff, err := parseDifferenceResult(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if diff.State == nil || diff.State.Date != 10 || diff.State.Seq != 20 {
		t.Fatalf("unexpected diff state: %+v", diff.State)
	}
}

func TestParseDifferenceTextMessageWithBotCommandEntity(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorDifference)
	b.putInt(constructorVector)
	b.putInt(1)
	b.putRaw(encodeTestMessage(7, 100, 100, "/start"))
	diff, err := parseDifferenceResult(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(diff.Messages))
	}
	msg := diff.Messages[0]
	if msg.ID != 7 || msg.FromID != 100 || msg.ChatID != 100 || msg.Text != "/start" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestParseDifferenceUpdateNewChannelMessage(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorDifference)
	b.putInt(constructorVector) // new_messages
	b.putInt(0)
	b.putInt(constructorVector) // encrypted_messages
	b.putInt(0)
	b.putInt(constructorVector) // other_updates
	b.putInt(1)
	b.putInt(constructorUpdateNewChannelMessage)
	b.putRaw(encodeTestChannelMessage(99, 12345, 777, "/ping"))
	b.putInt(44)                // pts
	b.putInt(1)                 // pts_count
	b.putInt(constructorVector) // chats
	b.putInt(0)
	b.putInt(constructorVector) // users
	b.putInt(0)
	b.putInt(constructorUpdatesState)
	b.putInt(44)
	b.putInt(0)
	b.putInt(123)
	b.putInt(9)
	b.putInt(0)

	diff, err := parseDifferenceResult(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(diff.Messages))
	}
	msg := diff.Messages[0]
	if msg.ID != 99 || msg.FromID != 777 || msg.ChatID != 12345 || msg.ChatKind != "channel" || msg.Text != "/ping" {
		t.Fatalf("unexpected channel message: %+v", msg)
	}
}

func encodeTestChannelMessage(id int32, channelID, fromUserID int64, text string) []byte {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, constructorMessage)
	b = binary.LittleEndian.AppendUint32(b, 1<<8|1<<7) // from_id + entities
	b = binary.LittleEndian.AppendUint32(b, 0)         // flags2
	b = binary.LittleEndian.AppendUint32(b, uint32(id))
	b = binary.LittleEndian.AppendUint32(b, constructorPeerUser)
	b = binary.LittleEndian.AppendUint64(b, uint64(fromUserID))
	b = binary.LittleEndian.AppendUint32(b, constructorPeerChannel)
	b = binary.LittleEndian.AppendUint64(b, uint64(channelID))
	b = binary.LittleEndian.AppendUint32(b, 123) // date
	var tb tlBuffer
	tb.buf = b
	tb.putString(text)
	tb.putInt(constructorVector)
	tb.putInt(1)
	tb.putInt(constructorMessageEntityBotCommand)
	tb.putInt(0)
	tb.putInt(uint32(len(text)))
	return tb.bytes()
}
