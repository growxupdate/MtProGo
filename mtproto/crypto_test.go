package mtproto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestProductionRSAFingerprint(t *testing.T) {
	key := mustProductionKey()
	got := key.Fingerprint
	const want uint64 = 0xd09d1d85de64fd85
	if got != want {
		t.Fatalf("fingerprint = %016x, want %016x", got, want)
	}
}

func TestAESIGERoundTrip(t *testing.T) {
	key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	iv, _ := hex.DecodeString("202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f")
	plain := bytes.Repeat([]byte{0x42}, 64)
	ciphertext, err := aesIGEEncrypt(plain, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := aesIGEDecrypt(ciphertext, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plain) {
		t.Fatal("decrypt(encrypt(plain)) mismatch")
	}
}

func TestTempAESKeysLength(t *testing.T) {
	var newNonce [32]byte
	var serverNonce [16]byte
	for i := range newNonce {
		newNonce[i] = byte(i)
	}
	for i := range serverNonce {
		serverNonce[i] = byte(32 + i)
	}
	key, iv := tempAESKeys(newNonce, serverNonce)
	if len(key) != 32 || len(iv) != 32 {
		t.Fatalf("key/iv lengths = %d/%d, want 32/32", len(key), len(iv))
	}
}
