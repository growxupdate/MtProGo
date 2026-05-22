package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mtprogo "github.com/growxupdate/MtProGo"
	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

func ask(label string) string {
	fmt.Print(label)
	r := bufio.NewReader(os.Stdin)
	v, err := r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(v)
}

func main() {
	fmt.Println("MtProGo V14.1 MTProto chat helper example")
	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}

	bot := mtprogo.Must(mtprogo.NewMTProtoBot(
		mtprogo.MTProtoBotConfig{
			APIID:        int(apiID64),
			APIHash:      apiHash,
			Token:        botToken,
			PollInterval: 2 * time.Second,
			PTSLimit:     100,
		},
		mtprogo.WithSessionFile("bot.session"),
		mtprogo.WithMessageCacheCommands("start", "ping", "chatid"),
		mtprogo.WithPeerCacheSize(5000),
	))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, fmt.Sprintf("MtProGo V14.1 ready in %s chat", m.ChatType))
	})

	bot.OnMessage(filters.And(filters.Command("ping"), filters.Private), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "private pong")
	})

	bot.OnMessage(filters.And(filters.Command("ping"), filters.Group), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "group pong")
	})

	bot.OnMessage(filters.And(filters.Command("ping"), filters.Channel), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "channel/supergroup pong")
	})

	bot.OnMessage(filters.Command("chatid"), func(ctx context.Context, m *updates.Message) error {
		return bot.SendMessage(ctx, m.ChatID, fmt.Sprintf("chat_id=%d type=%s", m.ChatID, m.ChatType))
	})

	if err := bot.Run(context.Background()); err != nil {
		panic(err)
	}
}
