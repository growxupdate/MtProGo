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
	value, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(value)
}

func main() {
	fmt.Println("MtProGo V8 Real Bot Example")
	fmt.Println("Version:", mtprogo.Version)
	fmt.Println()

	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	token := ask("Enter Bot Token: ")

	apiID, err := strconv.Atoi(apiIDText)
	if err != nil {
		panic("invalid API ID")
	}

	bot := mtprogo.Must(mtprogo.NewBot(mtprogo.BotConfig{
		APIID:   apiID,
		APIHash: apiHash,
		Token:   token,
		Session: "bot.session",
	}))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		fmt.Println("/start from", m.FromID, "in chat", m.ChatID)
		return m.Reply(ctx, "Hello from MtProGo V8 🚀")
	})

	fmt.Println("Send /start to your bot now.")
	mtprogo.Must0(bot.Run(context.Background()))
}
