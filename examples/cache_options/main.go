package main

import (
	"context"
	"fmt"

	mtprogo "github.com/growxupdate/MtProGo"
	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

func main() {
	client := mtprogo.Must(mtprogo.New(
		12345,
		"test_hash",
		mtprogo.WithUpdates(true),
		mtprogo.WithMessageCacheSize(3), // keep only last 3 messages in RAM
		mtprogo.WithPeerCacheSize(2),    // keep only last 2 peers in RAM
		mtprogo.WithUpdateQueueSize(64),
	))

	client.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		fmt.Println("handler:", m.Text)
		return nil
	})

	for i := 1; i <= 5; i++ {
		_ = client.Dispatcher().DispatchMessage(context.Background(), &updates.Message{
			ID:     i,
			ChatID: int64(1000 + i),
			FromID: int64(2000 + i),
			Text:   fmt.Sprintf("/start %d", i),
		})
	}

	snapshot := client.CacheSnapshot()
	fmt.Printf("messages cached: %d/%d\n", snapshot.MessageCount, snapshot.MessageLimit)
	fmt.Printf("peers cached: %d/%d\n", snapshot.PeerCount, snapshot.PeerLimit)

	for _, msg := range client.Dispatcher().MessageCache().All() {
		fmt.Printf("cached message: id=%d text=%q\n", msg.ID, msg.Text)
	}
}
