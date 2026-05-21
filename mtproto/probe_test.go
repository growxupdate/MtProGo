package mtproto

import "testing"

func TestParseResPQRejectsWrongConstructor(t *testing.T) {
	_, err := parseResPQ([]byte{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUint64FromBytes(t *testing.T) {
	if got := Uint64FromBytes([]byte{0x01, 0x02}); got != 0x0102 {
		t.Fatalf("unexpected value: %x", got)
	}
}
