package mtproto

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestPBKDF2HMACSHA512Vector(t *testing.T) {
	got := pbkdf2HMACSHA512([]byte("password"), []byte("salt"), 1, 64)
	want, _ := hex.DecodeString("867f70cf1ade02cff3752599a3a53dc4af34c7a669815ae5d513554e1c8cf252c02d470a285a0501bad999bfe943c08f050235d7d68b1da55e63f73b60a57fce")
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected pbkdf2 vector\n got %x\nwant %x", got, want)
	}
}

func TestMakeAuthCheckPasswordQuery(t *testing.T) {
	proof := &SRPProof{SRPID: 123, A: bytes.Repeat([]byte{1}, 256), M1: bytes.Repeat([]byte{2}, sha256.Size)}
	q := makeAuthCheckPasswordQuery(proof)
	if len(q) == 0 {
		t.Fatal("empty query")
	}
	r := newTLReader(q)
	method, err := r.int()
	if err != nil || method != constructorAuthCheckPassword {
		t.Fatalf("method = 0x%08x, err=%v", method, err)
	}
	check, err := r.int()
	if err != nil || check != constructorInputCheckPasswordSRP {
		t.Fatalf("input = 0x%08x, err=%v", check, err)
	}
	srpID, err := r.long()
	if err != nil || srpID != 123 {
		t.Fatalf("srpID = %d, err=%v", srpID, err)
	}
	A, err := r.bytes()
	if err != nil || !bytes.Equal(A, proof.A) {
		t.Fatalf("A mismatch err=%v", err)
	}
	M1, err := r.bytes()
	if err != nil || !bytes.Equal(M1, proof.M1) {
		t.Fatalf("M1 mismatch err=%v", err)
	}
	if len(r.remaining()) != 0 {
		t.Fatalf("unexpected remaining bytes: %d", len(r.remaining()))
	}
}
