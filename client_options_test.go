package mtprogo

import "testing"

func TestClientCacheOptions(t *testing.T) {
	client, err := New(1, "hash", WithUpdates(false), WithMessageCacheSize(7), WithPeerCacheSize(8), WithUpdateQueueSize(9))
	if err != nil {
		t.Fatal(err)
	}
	if client.UpdatesEnabled() {
		t.Fatal("updates should be disabled")
	}
	if client.MessageCacheSize() != 7 || client.PeerCacheSize() != 8 || client.UpdateQueueSize() != 9 {
		t.Fatalf("unexpected sizes: %d %d %d", client.MessageCacheSize(), client.PeerCacheSize(), client.UpdateQueueSize())
	}
}
