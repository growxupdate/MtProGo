package main

import (
	"bufio"
	"context"
	"errors"
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
	fmt.Println("MtProGo V9 pure MTProto user account login")
	fmt.Println("This example sends auth.sendCode, calls auth.signIn, supports SESSION_PASSWORD_NEEDED with SRP 2FA, then checks updates.getState.")
	fmt.Println()

	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	phone := ask("Enter Phone Number (+countrycode...): ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}
	apiID := int(apiID64)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	client, sent, err := mtproto.AuthSendCodeDefault(ctx, apiID, apiHash, phone)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println()
	fmt.Println("auth.sendCode OK")
	fmt.Println("Address:", client.DC().Address)
	fmt.Println("DC ID:", client.DC().ID)
	fmt.Println("SentCode constructor:", sent.ConstructorHex(), mtproto.ConstructorName(sent.Constructor))
	fmt.Println("Code type:", sent.CodeTypeHex(), mtproto.ConstructorName(sent.CodeTypeConstructor))
	fmt.Println("Phone code hash:", sent.PhoneCodeHash)
	if sent.Timeout > 0 {
		fmt.Println("Timeout:", sent.Timeout)
	}
	if sent.PhoneCodeHash == "" {
		fmt.Println("No phone_code_hash was returned; the account may already be authorized or a special login flow was returned.")
		return
	}

	code := ask("Enter Login Code: ")
	loginResult, err := client.AuthSignIn(ctx, apiID, phone, sent.PhoneCodeHash, code)
	if err != nil {
		var rpcErr *mtproto.RPCError
		if errors.As(err, &rpcErr) && strings.Contains(rpcErr.Message, "SESSION_PASSWORD_NEEDED") {
			fmt.Println("Telegram says SESSION_PASSWORD_NEEDED.")
			password := ask("Enter 2FA Password: ")

			params, getPasswordResult, err := client.AccountGetPassword(ctx)
			if err != nil {
				panic(err)
			}
			fmt.Println()
			fmt.Println("account.getPassword OK")
			fmt.Println("Result constructor:", getPasswordResult.ConstructorHex(), mtproto.ConstructorName(getPasswordResult.Constructor))
			fmt.Println("Has password:", params.HasPassword)
			fmt.Println("Hint:", params.Hint)
			fmt.Println("SRP ID:", params.SRPID)
			fmt.Println("KDF algo:", params.CurrentAlgo.ConstructorHex(), mtproto.ConstructorName(params.CurrentAlgo.Constructor))
			fmt.Println("SRP B bytes:", len(params.SRPB))

			loginResult, err = client.AuthCheckPassword(ctx, params, password)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	fmt.Println()
	fmt.Println("Account authorization OK")
	fmt.Println("Result constructor:", loginResult.ConstructorHex(), mtproto.ConstructorName(loginResult.Constructor))
	fmt.Println("Result bytes:", len(loginResult.Body))

	state, stateResult, err := client.UpdatesGetState(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println()
	fmt.Println("updates.getState OK")
	fmt.Println("Result constructor:", stateResult.ConstructorHex(), mtproto.ConstructorName(stateResult.Constructor))
	fmt.Println("PTS:", state.PTS)
	fmt.Println("QTS:", state.QTS)
	fmt.Println("Date:", state.Date)
	fmt.Println("Seq:", state.Seq)
	fmt.Println("Unread count:", state.UnreadCount)
}
