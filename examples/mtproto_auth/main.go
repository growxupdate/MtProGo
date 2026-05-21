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

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Telegram DC address, or press Enter to try defaults: ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	var (
		result *mtproto.AuthKeyResult
		err    error
	)
	if address == "" {
		result, err = mtproto.AuthKeyDefault(ctx)
	} else {
		fmt.Print("DC ID for this address: ")
		dcText, _ := reader.ReadString('\n')
		dcText = strings.TrimSpace(dcText)
		dcID, parseErr := strconv.Atoi(dcText)
		if parseErr != nil || dcID == 0 {
			panic("valid DC ID is required")
		}
		result, err = mtproto.AuthKey(ctx, mtproto.DCOption{ID: dcID, Address: address})
	}
	if err != nil {
		panic(err)
	}

	fmt.Println("Pure MTProto auth key OK")
	fmt.Println("Address:", result.Address)
	fmt.Println("DC ID:", result.DCID)
	fmt.Printf("RSA fingerprint: %016x\n", result.RSAFingerprint)
	fmt.Println("Auth key bytes:", len(result.AuthKey))
	fmt.Printf("Auth key ID: %016x\n", result.AuthKeyID)
	fmt.Println("Server salt:", result.ServerSalt)
	fmt.Println("Server time offset:", result.TimeOffset)
}
