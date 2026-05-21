package updates

import (
	"context"
	"testing"
)

func TestMessageCacheLimit(t *testing.T) {
	cache := NewMessageCache(2)
	cache.Add(&Message{ID: 1, Text: "one"})
	cache.Add(&Message{ID: 2, Text: "two"})
	cache.Add(&Message{ID: 3, Text: "three"})
	items := cache.All()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != 2 || items[1].ID != 3 {
		t.Fatalf("unexpected order: %+v", items)
	}
}

func TestDispatcherUpdatesDisabledStillCaches(t *testing.T) {
	d := NewDispatcher(WithUpdates(false), WithMessageCacheSize(1), WithPeerCacheSize(1))
	called := false
	d.OnMessage(nil, func(context.Context, *Message) error { called = true; return nil })
	if err := d.DispatchMessage(context.Background(), &Message{ID: 9, ChatID: 1, FromID: 1, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("handler should not be called when updates are disabled")
	}
	snap := d.CacheSnapshot()
	if snap.MessageCount != 1 || snap.PeerCount != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestPeerCacheLimit(t *testing.T) {
	cache := NewPeerCache(1)
	cache.Put(Peer{ID: 1, Kind: "user"})
	cache.Put(Peer{ID: 2, Kind: "user"})
	if _, ok := cache.Get(1); ok {
		t.Fatal("old peer should be evicted")
	}
	if _, ok := cache.Get(2); !ok {
		t.Fatal("new peer should exist")
	}
}
