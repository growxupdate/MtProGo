package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/growxupdate/MtProGo/mtproto"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Telegram DC address, or press Enter to try defaults: ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var (
		result *mtproto.ProbeResult
		err    error
	)
	if address == "" {
		result, err = mtproto.ProbeDefault(ctx)
	} else {
		result, err = mtproto.Probe(ctx, address)
	}
	if err != nil {
		panic(err)
	}

	fmt.Println("Pure MTProto probe OK")
	fmt.Println("Address:", result.Address)
	fmt.Println("PQ hex:", result.PQHex())
	fmt.Println("P:", result.PHex())
	fmt.Println("Q:", result.QHex())
	fmt.Printf("RSA fingerprints: %x\n", result.RSAFingerprints)
}
