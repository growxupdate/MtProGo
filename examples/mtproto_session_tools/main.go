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
	fmt.Println("MtProGo V13 MTProto bot session tools")
	fmt.Println("Modes: export, clear, check")
	mode := ask("Mode [export]: ")
	if mode == "" {
		mode = "export"
	}
	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")
	sessionPath := ask("Session file [bot.session]: ")
	if sessionPath == "" {
		sessionPath = "bot.session"
	}
	password := ask("Encrypted session password optional: ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}

	opts := []mtprogo.Option{mtprogo.WithUpdates(false)}
	if password != "" {
		opts = append(opts, mtprogo.WithEncryptedSessionFile(sessionPath, password))
	} else {
		opts = append(opts, mtprogo.WithSessionFile(sessionPath))
	}
	bot := mtprogo.Must(mtprogo.NewMTProtoBot(mtprogo.MTProtoBotConfig{
		APIID:        int(apiID64),
		APIHash:      apiHash,
		Token:        botToken,
		PollInterval: time.Second,
	}, opts...))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	switch strings.ToLower(mode) {
	case "clear":
		if err := bot.ClearSession(ctx); err != nil {
			panic(err)
		}
		fmt.Println("Session cleared:", sessionPath)
		return
	case "check", "export":
		if err := bot.Login(ctx); err != nil {
			panic(err)
		}
		defer bot.Close()
		if mode == "check" {
			fmt.Println("Session check/login OK")
			return
		}
		encoded, err := bot.ExportStringSession()
		if err != nil {
			panic(err)
		}
		fmt.Println("String session:")
		fmt.Println(encoded)
	default:
		panic("unknown mode")
	}
}
