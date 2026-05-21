package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestMTProto2AESKeyIV(t *testing.T) {
	authKey := make([]byte, 256)
	msgKey := make([]byte, 16)
	for i := range authKey {
		authKey[i] = byte(i)
	}
	for i := range msgKey {
		msgKey[i] = byte(i + 3)
	}
	key, iv := mtproto2AESKeyIV(authKey, msgKey, 0)
	if len(key) != 32 || len(iv) != 32 {
		t.Fatalf("unexpected key/iv length: %d/%d", len(key), len(iv))
	}
	key2, iv2 := mtproto2AESKeyIV(authKey, msgKey, 8)
	if equalBytes(key, key2) || equalBytes(iv, iv2) {
		t.Fatal("client and server key/iv should differ")
	}
}

func TestMakeHelpGetConfigQuery(t *testing.T) {
	direct := makeHelpGetConfigQuery(0)
	if got := binary.LittleEndian.Uint32(direct[:4]); got != constructorHelpGetConfig {
		t.Fatalf("direct constructor = 0x%08x", got)
	}
	wrapped := makeHelpGetConfigQuery(12345)
	if got := binary.LittleEndian.Uint32(wrapped[:4]); got != constructorInvokeWithLayer {
		t.Fatalf("wrapped constructor = 0x%08x", got)
	}
}
