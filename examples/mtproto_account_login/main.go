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

	mtprogo "github.com/growxupdate/MtProGo"
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

func defaultValue(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func main() {
	fmt.Println("MtProGo V13 pure MTProto user account login with persistent session")
	fmt.Println("First run logs in and saves account.session. Next run loads it without phone code.")
	fmt.Println()

	apiIDText := ask("Enter API ID: ")
	apiHash := ask("Enter API Hash: ")
	sessionPath := defaultValue(ask("Session file [account.session]: "), "account.session")
	encPassword := ask("Encrypt session password optional [empty = plain file]: ")
	stringSession := ask("Paste string session optional [empty = use file/login]: ")

	apiID64, err := strconv.ParseInt(apiIDText, 10, 32)
	if err != nil || apiID64 <= 0 {
		panic("invalid API ID")
	}
	apiID := int(apiID64)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	var store mtprogo.SessionStore
	name := "account"
	if stringSession != "" {
		sess, err := mtproto.ParseStringSession(stringSession)
		if err != nil {
			panic(err)
		}
		data, err := sess.MarshalBinary()
		if err != nil {
			panic(err)
		}
		store = mtprogo.MemorySession()
		if err := store.Save(ctx, mtprogo.SessionData{Name: name, Data: data}); err != nil {
			panic(err)
		}
	} else if encPassword != "" {
		store = mtprogo.EncryptedFileSession(sessionPath, encPassword)
	} else {
		store = mtprogo.FileSession(sessionPath)
	}

	if data, ok, err := store.Load(ctx, name); err != nil {
		fmt.Println("saved session load failed; will login again:", err)
	} else if ok {
		sess, err := mtproto.ParseSession(data.Data)
		if err != nil {
			fmt.Println("saved session parse failed; will login again:", err)
		} else {
			client, err := mtproto.DialEncryptedFromSession(ctx, sess)
			if err != nil {
				fmt.Println("saved session connect failed; will login again:", err)
			} else {
				defer client.Close()
				state, stateResult, err := client.UpdatesGetState(ctx)
				if err != nil {
					fmt.Println("saved session check failed; clear the session or login again:", err)
				} else {
					fmt.Println("Loaded account session OK")
					fmt.Println("Address:", client.DC().Address)
					fmt.Println("DC ID:", client.DC().ID)
					fmt.Println("updates.getState:", stateResult.ConstructorHex(), mtproto.ConstructorName(stateResult.Constructor))
					fmt.Println("PTS:", state.PTS, "QTS:", state.QTS, "Date:", state.Date, "Seq:", state.Seq)
					encoded, err := client.ExportSession("user").EncodeString()
					if err == nil {
						fmt.Println("String session:", encoded)
					}
					return
				}
			}
		}
	}

	phone := ask("Enter Phone Number (+countrycode...): ")
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
			fmt.Println("Hint:", params.Hint)
			fmt.Println("SRP ID:", params.SRPID)

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

	state, stateResult, err := client.UpdatesGetState(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println()
	fmt.Println("updates.getState OK")
	fmt.Println("Result constructor:", stateResult.ConstructorHex(), mtproto.ConstructorName(stateResult.Constructor))
	fmt.Println("PTS:", state.PTS, "QTS:", state.QTS, "Date:", state.Date, "Seq:", state.Seq)

	sess := client.ExportSession("user")
	data, err := sess.MarshalBinary()
	if err != nil {
		panic(err)
	}
	if err := store.Save(ctx, mtprogo.SessionData{Name: name, Data: data}); err != nil {
		panic(err)
	}
	fmt.Println("Session saved:", sessionPath)
	encoded, err := sess.EncodeString()
	if err == nil {
		fmt.Println("String session:", encoded)
	}
}
