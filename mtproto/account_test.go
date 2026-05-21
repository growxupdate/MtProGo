package mtproto

import "testing"

func TestMakeAuthSendCodeQuery(t *testing.T) {
	q := makeAuthSendCodeQuery(12345, "hash", "+10000000000")
	if len(q) == 0 {
		t.Fatal("empty query")
	}
}

func TestMakeAuthSignInQuery(t *testing.T) {
	q := makeAuthSignInQuery(12345, "+10000000000", "codehash", "12345")
	if len(q) == 0 {
		t.Fatal("empty query")
	}
}

func TestParseSentCodeRejectsWrongConstructor(t *testing.T) {
	_, err := parseSentCodeResult("+1", []byte{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error")
	}
}
