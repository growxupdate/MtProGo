package mtproto

import (
	"encoding/binary"
	"testing"
)

func TestMakeImportBotAuthorizationQueryWrapsLayer(t *testing.T) {
	q := makeImportBotAuthorizationQuery(12345, "hash", "123:token")
	if got := binary.LittleEndian.Uint32(q[:4]); got != constructorInvokeWithLayer {
		t.Fatalf("constructor = %#x, want invokeWithLayer", got)
	}
	if got := binary.LittleEndian.Uint32(q[4:8]); got != uint32(defaultAPILayer) {
		t.Fatalf("layer = %d, want %d", got, defaultAPILayer)
	}
}

func TestMakeMessagesSendMessageQuery(t *testing.T) {
	q, err := makeMessagesSendMessageQuery(InputPeerUser(10, 20), "hello")
	if err != nil {
		t.Fatal(err)
	}
	r := newTLReader(q)
	constructor, err := r.int()
	if err != nil {
		t.Fatal(err)
	}
	if constructor != constructorMessagesSendMessage {
		t.Fatalf("constructor = %#x", constructor)
	}
	flags, err := r.int()
	if err != nil {
		t.Fatal(err)
	}
	if flags != 0 {
		t.Fatalf("flags = %d", flags)
	}
	peerConstructor, err := r.int()
	if err != nil {
		t.Fatal(err)
	}
	if peerConstructor != constructorInputPeerUser {
		t.Fatalf("peer constructor = %#x", peerConstructor)
	}
	uid, _ := r.long()
	ah, _ := r.long()
	if uid != 10 || ah != 20 {
		t.Fatalf("peer = %d/%d", uid, ah)
	}
	text, err := r.string()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Fatalf("text = %q", text)
	}
}

func TestInputPeerValidation(t *testing.T) {
	_, err := makeMessagesSendMessageQuery(InputPeerUser(10, 0), "hi")
	if err == nil {
		t.Fatal("expected error for missing access hash")
	}
	_, err = makeMessagesSendMessageQuery(InputPeerChat(123), "hi")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMakeMessagesEditAndDeleteQueries(t *testing.T) {
	ed, err := makeMessagesEditMessageQuery(InputPeerChat(123), 7, "edited")
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(ed[:4]); got != constructorMessagesEditMessage {
		t.Fatalf("edit constructor = %#x", got)
	}
	del := makeMessagesDeleteMessagesQuery(true, 7, 8)
	if got := binary.LittleEndian.Uint32(del[:4]); got != constructorMessagesDeleteMessages {
		t.Fatalf("delete constructor = %#x", got)
	}
	if flags := binary.LittleEndian.Uint32(del[4:8]); flags != 1 {
		t.Fatalf("delete flags = %d", flags)
	}
}
