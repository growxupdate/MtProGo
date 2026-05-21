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

func main() {
	fmt.Println("MtProGo V11 pure MTProto bot authorization")
	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	botToken := ask("Enter Bot Token: ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, result, err := mtproto.ImportBotAuthorizationDefault(ctx, int(apiID64), apiHash, botToken)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("Pure MTProto bot authorization OK")
	fmt.Println("Address:", result.Address)
	fmt.Println("DC ID:", result.DCID)
	fmt.Println("Auth key ID:", result.AuthKeyIDHex())
	fmt.Println("Server salt:", result.ServerSalt)
	fmt.Println("Session ID:", result.SessionID)
	fmt.Println("Result constructor:", result.ConstructorHex(), mtproto.ConstructorName(result.Constructor))
	fmt.Println("Result bytes:", len(result.Body))
}
