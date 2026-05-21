package updates

import (
	"context"
	"testing"
)

func TestDefaultCacheStoresOnlyMatchedMessages(t *testing.T) {
	d := NewDispatcher(WithMessageCacheSize(10), WithPeerCacheSize(10))
	d.OnMessage(func(m *Message) bool { return m.Text == "/start" }, func(context.Context, *Message) error { return nil })
	if err := d.DispatchMessage(context.Background(), &Message{ID: 1, ChatID: 10, FromID: 10, Text: "noise"}); err != nil {
		t.Fatal(err)
	}
	if err := d.DispatchMessage(context.Background(), &Message{ID: 2, ChatID: 10, FromID: 10, Text: "/start"}); err != nil {
		t.Fatal(err)
	}
	snap := d.CacheSnapshot()
	if snap.MessageCount != 1 || snap.PeerCount != 1 || snap.CacheMode != "matched" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestCacheAllMessages(t *testing.T) {
	d := NewDispatcher(WithMessageCacheSize(10), WithMessageCacheMode(CacheAllMessages))
	if err := d.DispatchMessage(context.Background(), &Message{ID: 1, Text: "noise"}); err != nil {
		t.Fatal(err)
	}
	if got := d.CacheSnapshot().MessageCount; got != 1 {
		t.Fatalf("message count = %d, want 1", got)
	}
}

func TestCacheFilteredMessages(t *testing.T) {
	d := NewDispatcher(
		WithMessageCacheSize(10),
		WithMessageCacheFilter(func(m *Message) bool { return m.Text == "/ping" }),
	)
	_ = d.DispatchMessage(context.Background(), &Message{ID: 1, Text: "/start"})
	_ = d.DispatchMessage(context.Background(), &Message{ID: 2, Text: "/ping"})
	items := d.MessageCache().All()
	if len(items) != 1 || items[0].Text != "/ping" {
		t.Fatalf("unexpected cached items: %+v", items)
	}
}
