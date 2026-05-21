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
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(text)
}

func main() {
	fmt.Println("MtProGo V6 encrypted MTProto help.getConfig test")
	fmt.Println("This creates a temporary auth key, sends an encrypted request, and prints the response constructor.")
	fmt.Println()
	apiIDText := ask("Enter API ID, or press Enter to send direct help.getConfig: ")
	apiID := 0
	if apiIDText != "" {
		parsed, err := strconv.Atoi(apiIDText)
		if err != nil {
			panic("invalid API ID")
		}
		apiID = parsed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := mtproto.HelpGetConfigDefault(ctx, apiID)
	if err != nil {
		panic(err)
	}
	fmt.Println()
	fmt.Println("Encrypted MTProto help.getConfig OK")
	fmt.Println("Address:", result.Address)
	fmt.Println("DC ID:", result.DCID)
	fmt.Println("Auth key ID:", result.AuthKeyIDHex())
	fmt.Println("Server salt:", result.ServerSalt)
	fmt.Println("Session ID:", result.SessionID)
	fmt.Println("Request msg ID:", result.RequestMessageID)
	fmt.Println("Response msg ID:", result.ResponseMessageID)
	fmt.Println("Result constructor:", result.ConstructorHex())
	fmt.Println("Gzip packed:", result.GzipPacked)
	fmt.Println("Result bytes:", len(result.Body))
}
