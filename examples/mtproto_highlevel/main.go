package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	mtprogo "github.com/growxupdate/MtProGo"
	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

func ask(label string) string {
	fmt.Print(label)
	text, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(text)
}

func main() {
	fmt.Println("MtProGo V17 high-level MTProto helper example")
	apiID64, err := strconv.ParseInt(ask("Enter API ID: "), 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")

	bot := mtprogo.Must(mtprogo.NewMTProtoBot(
		mtprogo.MTProtoBotConfig{APIID: int(apiID64), APIHash: apiHash, Token: botToken},
		mtprogo.WithSessionFile("bot.session"),
		mtprogo.WithUpdates(true),
		mtprogo.WithDebug(false),
	))

	bot.OnMessageClient(filters.Command("start"), func(ctx context.Context, c *mtprogo.MTProtoBot, m *updates.Message) error {
		me, err := c.GetMe(ctx)
		if err != nil {
			return m.Reply(ctx, "Hello from MtProGo V17")
		}
		name := me.DisplayName()
		if name == "" {
			name = "MtProGo"
		}
		return m.Reply(ctx, "Hello from "+name+" 🚀")
	})

	bot.OnMessage(filters.Command("ping"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "pong")
	})

	bot.OnMessage(filters.Command("chatid"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, fmt.Sprintf("chat_id=%d raw_chat_id=%d type=%s from_id=%d", m.ChatID, m.RawChatID, m.ChatType, m.FromID))
	})

	fmt.Println("Send /start, /ping, or /chatid to your bot in private or group chats.")
	if err := bot.Run(context.Background()); err != nil {
		panic(err)
	}
}
