package mtproto

import (
	"bytes"
	"testing"
)

func TestSessionStringRoundTrip(t *testing.T) {
	s := &Session{
		Version:    1,
		Kind:       "bot",
		DCID:       5,
		Address:    "91.108.56.130:443",
		AuthKey:    bytes.Repeat([]byte{7}, 256),
		AuthKeyID:  123456789,
		ServerSalt: 42,
		TimeOffset: -3,
	}
	encoded, err := s.EncodeString()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseStringSession(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.DCID != s.DCID || parsed.Address != s.Address || parsed.AuthKeyID != s.AuthKeyID || !bytes.Equal(parsed.AuthKey, s.AuthKey) {
		t.Fatalf("bad session round trip: %+v", parsed)
	}
}
