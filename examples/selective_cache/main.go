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
		mtprogo.WithMessageCacheSize(1000),
		// Only cache these commands. Random text/noise updates are handled or ignored
		// without being kept in RAM.
		mtprogo.WithMessageCacheCommands("start", "ping", "help", "queue"),
	))

	client.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		return nil
	})
	client.OnMessage(filters.Command("ping"), func(ctx context.Context, m *updates.Message) error {
		return nil
	})
	client.OnMessage(filters.Command("help"), func(ctx context.Context, m *updates.Message) error {
		return nil
	})
	client.OnMessage(filters.Command("queue"), func(ctx context.Context, m *updates.Message) error {
		return nil
	})

	ctx := context.Background()
	for i := 1; i <= 10000; i++ {
		text := "hello noise"
		switch i {
		case 7:
			text = "/start"
		case 100:
			text = "/ping"
		case 777:
			text = "/help"
		case 9999:
			text = "/queue"
		}
		_ = client.Dispatcher().DispatchMessage(ctx, &updates.Message{ID: i, ChatID: int64(i), FromID: int64(i), Text: text})
	}

	s := client.CacheSnapshot()
	fmt.Printf("cache mode=%s messages=%d/%d peers=%d/%d\n", s.CacheMode, s.MessageCount, s.MessageLimit, s.PeerCount, s.PeerLimit)
	fmt.Println("Only /start, /ping, /help, and /queue were cached; 9996 noisy messages were not kept in RAM.")
}
