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
	text, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(text)
}

func main() {
	fmt.Println("MtProGo V18 pure MTProto media example")
	apiID64, err := strconv.ParseInt(ask("Enter API ID: "), 10, 32)
	if err != nil {
		panic(err)
	}
	apiHash := ask("Enter API Hash: ")
	token := ask("Enter Bot Token: ")
	docPath := ask("Document path to send on /doc [README.md]: ")
	if docPath == "" {
		docPath = "README.md"
	}
	photoPath := ask("Photo path to send on /photo [optional]: ")

	bot := mtprogo.Must(mtprogo.NewMTProtoBot(
		mtprogo.MTProtoBotConfig{APIID: int(apiID64), APIHash: apiHash, Token: token},
		mtprogo.WithSessionFile("media_bot.session"),
		mtprogo.WithUpdates(true),
		mtprogo.WithMessageCacheCommands("start", "doc", "photo"),
		mtprogo.WithAutoReconnect(true),
		mtprogo.WithReconnectBackoff(time.Second, 30*time.Second),
		mtprogo.WithAutoFloodWait(true, 60*time.Second),
	))

	bot.OnMessageClient(filters.Command("start"), func(ctx context.Context, c *mtprogo.MTProtoBot, m *updates.Message) error {
		return m.Reply(ctx, "Send /doc to receive a document. Send /photo after providing a local photo path.")
	})

	bot.OnMessageClient(filters.Command("doc"), func(ctx context.Context, c *mtprogo.MTProtoBot, m *updates.Message) error {
		fmt.Println("/doc received, uploading", docPath)
		return c.SendDocument(ctx, m.ChatID, docPath, mtprogo.SendMediaOptions{Caption: "Document sent by MtProGo V18"})
	})

	bot.OnMessageClient(filters.Command("photo"), func(ctx context.Context, c *mtprogo.MTProtoBot, m *updates.Message) error {
		if photoPath == "" {
			return m.Reply(ctx, "No photo path was provided when the example started.")
		}
		fmt.Println("/photo received, uploading", photoPath)
		return c.SendPhoto(ctx, m.ChatID, photoPath, mtprogo.SendMediaOptions{Caption: "Photo sent by MtProGo V18"})
	})

	mtprogo.Must0(bot.Run(context.Background()))
}
