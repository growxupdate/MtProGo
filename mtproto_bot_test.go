package mtprogo

import (
	"testing"

	"github.com/growxupdate/MtProGo/mtproto"
	"github.com/growxupdate/MtProGo/updates"
)

func TestNewMTProtoBotOptions(t *testing.T) {
	bot, err := NewMTProtoBot(MTProtoBotConfig{APIID: 1, APIHash: "hash", Token: "123:abc"}, WithUpdates(false), WithMessageCacheSize(7), WithPeerCacheSize(9))
	if err != nil {
		t.Fatal(err)
	}
	if bot.dispatcher.UpdatesEnabled() {
		t.Fatal("updates should be disabled")
	}
	snap := bot.CacheSnapshot()
	if snap.MessageLimit != 7 || snap.PeerLimit != 9 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestMergeUpdateState(t *testing.T) {
	old := &mtproto.UpdatesState{PTS: 1, QTS: 2, Date: 3, Seq: 4}
	next := &mtproto.UpdatesState{PTS: 10, Date: 30}
	merged := mergeUpdateState(old, next)
	if merged.PTS != 10 || merged.QTS != 2 || merged.Date != 30 || merged.Seq != 4 {
		t.Fatalf("unexpected merged state: %+v", merged)
	}
}

func TestMTProtoBotTextMessageDispatchShape(t *testing.T) {
	bot, err := NewMTProtoBot(MTProtoBotConfig{APIID: 1, APIHash: "hash", Token: "123:abc"})
	if err != nil {
		t.Fatal(err)
	}
	bot.rememberPeers([]mtproto.PeerRef{{ID: 42, AccessHash: 99, Kind: "user"}})
	peer, ok := bot.getPeer(42)
	if !ok || peer.AccessHash != 99 {
		t.Fatalf("peer not stored: %+v ok=%v", peer, ok)
	}
	msg := bot.mtprotoTextMessageToUpdate(mtproto.TextMessage{ID: 5, ChatID: 42, FromID: 42, ChatKind: "user", FromKind: "user", Text: "/start"})
	if msg.ID != 5 || msg.ChatID != 42 || msg.FromID != 42 || msg.ChatType != updates.ChatPrivate || msg.Text != "/start" {
		t.Fatalf("bad message: %+v", msg)
	}
	if _, ok := msg.Raw.(mtproto.TextMessage); !ok {
		t.Fatalf("raw type = %T", msg.Raw)
	}
	if msg.ReplySender == nil {
		t.Fatal("reply sender missing")
	}
	if got, ok := bot.Dispatcher().PeerCache().Get(42); !ok || got.AccessHash != 99 || got.Kind != "user" {
		t.Fatalf("dispatcher peer not stored: %+v ok=%v", got, ok)
	}
	_ = updates.Message{}
}
