package mtproto

import "testing"

func TestParseUpdatesState(t *testing.T) {
	var b tlBuffer
	b.putInt(constructorUpdatesState)
	b.putInt(1)
	b.putInt(2)
	b.putInt(3)
	b.putInt(4)
	b.putInt(5)
	state, err := parseUpdatesState(b.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if state.PTS != 1 || state.QTS != 2 || state.Date != 3 || state.Seq != 4 || state.UnreadCount != 5 {
		t.Fatalf("bad state: %+v", state)
	}
}
