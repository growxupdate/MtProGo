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
	fmt.Println("MtProGo V16 pure MTProto bot updates with reliability hardening")
	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")
	sessionPath := ask("Session file [bot.session]: ")
	if sessionPath == "" {
		sessionPath = "bot.session"
	}
	encPassword := ask("Encrypt session password optional [empty = plain file]: ")
	stringSession := ask("Paste string session optional [empty = use file/login]: ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}

	opts := []mtprogo.Option{
		mtprogo.WithUpdates(true),
		mtprogo.WithMessageCacheSize(1000),
		mtprogo.WithPeerCacheSize(1000),
		mtprogo.WithMessageCacheCommands("start", "ping", "help", "chatid", "delete"),
		mtprogo.WithAutoReconnect(true),
		mtprogo.WithReconnectBackoff(time.Second, 30*time.Second),
		mtprogo.WithAutoFloodWait(true, 60*time.Second),
	}
	if stringSession != "" {
		opts = append(opts, mtprogo.WithStringSession(stringSession))
	} else if encPassword != "" {
		opts = append(opts, mtprogo.WithEncryptedSessionFile(sessionPath, encPassword))
	} else {
		opts = append(opts, mtprogo.WithSessionFile(sessionPath))
	}

	bot := mtprogo.Must(mtprogo.NewMTProtoBot(
		mtprogo.MTProtoBotConfig{
			APIID:        int(apiID64),
			APIHash:      apiHash,
			Token:        botToken,
			PollInterval: 2 * time.Second,
			PTSLimit:     100,
		},
		opts...,
	))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		fmt.Printf("/start received from %d in %s chat %d\n", m.FromID, m.ChatType, m.ChatID)
		if err := m.Reply(ctx, "Hello from MtProGo V16 over pure MTProto 🚀"); err != nil {
			return err
		}
		fmt.Println("Reply sent over MTProto")
		return nil
	})

	bot.OnMessage(filters.Command("ping"), func(ctx context.Context, m *updates.Message) error {
		if err := m.Reply(ctx, "pong"); err != nil {
			return err
		}
		fmt.Println("pong sent over MTProto")
		return nil
	})

	bot.OnMessage(filters.Command("help"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, "Commands: /start, /ping, /help, /chatid. Private chats, groups, and supergroups are parsed; V16 also persists update state and reconnects automatically.")
	})

	bot.OnMessage(filters.Command("chatid"), func(ctx context.Context, m *updates.Message) error {
		return m.Reply(ctx, fmt.Sprintf("chat_id=%d type=%s from_id=%d", m.ChatID, m.ChatType, m.FromID))
	})

	bot.OnMessage(filters.And(filters.Command("ping"), filters.Private), func(ctx context.Context, m *updates.Message) error {
		fmt.Println("private ping matched")
		return nil
	})

	fmt.Println("Send /start, /ping, /help or /chatid to your bot in private or group/channel contexts.")
	if err := bot.Run(context.Background()); err != nil {
		panic(err)
	}
}
