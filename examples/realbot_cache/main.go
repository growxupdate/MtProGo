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
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(text)
}

func askInt(label string, def int) int {
	value := ask(label)
	if value == "" {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		panic("invalid number")
	}
	return n
}

func askBool(label string, def bool) bool {
	value := strings.ToLower(ask(label))
	if value == "" {
		return def
	}
	return value == "true" || value == "yes" || value == "y" || value == "1"
}

func main() {
	fmt.Println("MtProGo V11 real bot with memory/cache options")
	fmt.Println("Send /start to your bot after it starts.")
	fmt.Println()

	token := ask("Enter Bot Token: ")
	updatesEnabled := askBool("Updates true/false [true]: ", true)
	messageCache := askInt("Message cache size [1000]: ", 1000)
	peerCache := askInt("Peer cache size [1000]: ", 1000)
	dropPending := askBool("Drop pending updates true/false [false]: ", false)

	bot := mtprogo.Must(mtprogo.NewBotWithOptions(
		mtprogo.BotConfig{Token: token, DropPendingUpdates: dropPending},
		mtprogo.WithUpdates(updatesEnabled),
		mtprogo.WithMessageCacheSize(messageCache),
		mtprogo.WithPeerCacheSize(peerCache),
	))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		s := bot.CacheSnapshot()
		fmt.Printf("/start from %d, cache messages=%d/%d peers=%d/%d\n", m.FromID, s.MessageCount, s.MessageLimit, s.PeerCount, s.PeerLimit)
		return m.Reply(ctx, fmt.Sprintf("MtProGo V11 🚀 cache=%d/%d", s.MessageCount, s.MessageLimit))
	})

	mtprogo.Must0(bot.Run(context.Background()))
}
