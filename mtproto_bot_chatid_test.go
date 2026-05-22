package mtprogo

import (
	"testing"

	"github.com/growxupdate/MtProGo/mtproto"
	"github.com/growxupdate/MtProGo/updates"
)

func TestMTProtoBotPublicChatIDForSupergroup(t *testing.T) {
	bot := &MTProtoBot{peers: make(map[int64]mtproto.PeerRef)}
	bot.rememberPeers([]mtproto.PeerRef{{ID: 1251846294, AccessHash: 99, Kind: "supergroup"}})

	msg := bot.mtprotoTextMessageToUpdate(mtproto.TextMessage{
		ID:       7,
		ChatID:   1251846294,
		ChatKind: "channel",
		FromID:   6521935712,
		FromKind: "user",
		Text:     "/start",
	})

	if msg.ChatType != updates.ChatSupergroup {
		t.Fatalf("chat type = %q, want %q", msg.ChatType, updates.ChatSupergroup)
	}
	if msg.ChatID != -1001251846294 {
		t.Fatalf("chat id = %d, want -1001251846294", msg.ChatID)
	}
	if msg.RawChatID != 1251846294 {
		t.Fatalf("raw chat id = %d, want 1251846294", msg.RawChatID)
	}
	peer, ok := bot.getPeer(msg.ChatID)
	if !ok {
		t.Fatal("expected public chat id lookup to resolve cached raw peer")
	}
	if peer.ID != 1251846294 || peer.AccessHash != 99 || peer.Kind != "supergroup" {
		t.Fatalf("unexpected peer: %+v", peer)
	}
}

func TestMTProtoBotPublicChatIDForBasicGroup(t *testing.T) {
	bot := &MTProtoBot{peers: make(map[int64]mtproto.PeerRef)}
	bot.rememberPeers([]mtproto.PeerRef{{ID: 777, Kind: "chat"}})
	msg := bot.mtprotoTextMessageToUpdate(mtproto.TextMessage{ID: 1, ChatID: 777, ChatKind: "chat", FromID: 42, FromKind: "user", Text: "/ping"})
	if msg.ChatID != -777 {
		t.Fatalf("chat id = %d, want -777", msg.ChatID)
	}
	if _, ok := bot.getPeer(msg.ChatID); !ok {
		t.Fatal("expected negative group chat id lookup to resolve raw peer")
	}
}
