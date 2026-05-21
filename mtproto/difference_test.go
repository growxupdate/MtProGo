package mtproto

import "testing"

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
