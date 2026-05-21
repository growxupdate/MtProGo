package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	mtprogo "github.com/growxupdate/MtProGo"
	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

func main() {
	apiID, _ := strconv.Atoi(os.Getenv("API_ID"))
	apiHash := os.Getenv("API_HASH")
	token := os.Getenv("BOT_TOKEN")

	bot := mtprogo.Must(mtprogo.NewBot(mtprogo.BotConfig{
		APIID:   apiID,
		APIHash: apiHash,
		Token:   token,
	}))

	bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
		fmt.Println("/start from", m.FromID)
		return m.Reply(ctx, "Hello from MtProGo V12 🚀")
	})

	mtprogo.Must0(bot.Run(context.Background()))
}
