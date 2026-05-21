package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	mtprogo "github.com/growxupdate/MtProGo"
	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

func ask(label string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(text)
}

func main() {
	token := ask("Enter Bot Token: ")

	bot := mtprogo.Must(mtprogo.NewBotWithOptions(
		mtprogo.BotConfig{Token: token, DropPendingUpdates: true},
		mtprogo.WithUpdates(true),
		mtprogo.WithMessageCacheSize(1000),
		mtprogo.WithPeerCacheSize(1000),
		// High-volume mode: keep RAM for useful commands only.
		mtprogo.WithMessageCacheCommands("start", "ping", "help", "queue"),
	))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		s := bot.CacheSnapshot()
		return m.Reply(ctx, fmt.Sprintf("MtProGo V11 🚀 cache=%s %d/%d", s.CacheMode, s.MessageCount, s.MessageLimit))
	})
	bot.OnMessage(filters.Command("ping"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "pong")
	})
	bot.OnMessage(filters.Command("help"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "Commands: /start /ping /help /queue")
	})
	bot.OnMessage(filters.Command("queue"), func(ctx context.Context, m *updates.Message) error {
		s := bot.CacheSnapshot()
		return m.Reply(ctx, fmt.Sprintf("Cached messages: %d/%d", s.MessageCount, s.MessageLimit))
	})

	mtprogo.Must0(bot.Run(context.Background()))
}
