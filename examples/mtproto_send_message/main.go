package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/growxupdate/MtProGo/mtproto"
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

func mustInt64(label string) int64 {
	v, err := strconv.ParseInt(ask(label), 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func main() {
	fmt.Println("MtProGo V12 pure MTProto sendMessage")
	fmt.Println("Note: inputPeerUser/inputPeerChannel require the Telegram access_hash.")
	fmt.Println("Bots normally learn access_hash values from MTProto updates or dialogs; that update loop is planned for the next update-loop milestone.")
	fmt.Println()

	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")
	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}

	peerKind := strings.ToLower(ask("Peer type (chat/user/channel/self): "))
	var peer mtproto.InputPeer
	switch peerKind {
	case "self":
		peer = mtproto.InputPeerSelf()
	case "chat":
		peer = mtproto.InputPeerChat(mustInt64("Chat ID: "))
	case "user":
		peer = mtproto.InputPeerUser(mustInt64("User ID: "), mustInt64("User access_hash: "))
	case "channel":
		peer = mtproto.InputPeerChannel(mustInt64("Channel ID: "), mustInt64("Channel access_hash: "))
	default:
		panic("unknown peer type")
	}
	text := ask("Message text: ")

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	client, loginResult, err := mtproto.ImportBotAuthorizationDefault(ctx, int(apiID64), apiHash, botToken)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	fmt.Println("Bot authorized:", loginResult.ConstructorHex(), mtproto.ConstructorName(loginResult.Constructor))

	result, err := client.MessagesSendMessage(ctx, peer, text)
	if err != nil {
		panic(err)
	}
	fmt.Println("messages.sendMessage OK")
	fmt.Println("Result constructor:", result.ConstructorHex(), mtproto.ConstructorName(result.Constructor))
	fmt.Println("Result bytes:", len(result.Body))
}
