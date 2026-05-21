# MtProGo

MtProGo is a clean Go Telegram client foundation built from scratch.

V9 is implemented with the Go standard library only. It does **not** import TDLib, gotd, tgbotapi, Pyrogram, Telethon, Kurigram, or any other Telegram client library.

## Status

Current V9 features:

- Real Telegram bot runtime using Telegram Bot API long polling
- `getMe`, `getUpdates`, `sendMessage`
- `OnMessage` handlers
- Command, regex, chat, user, and text filters
- `Message.Reply()` helper
- Pure Go MTProto `req_pq_multi` probe over TCP abridged
- Pure Go MTProto RSA/DH authorization key generation
- AES-IGE implementation
- MTProto RSA fingerprinting and RSA_PAD implementation
- Encrypted MTProto message packing and response parsing
- Encrypted `help.getConfig` example
- Pure MTProto bot authorization using `auth.importBotAuthorization`
- Raw encrypted `messages.sendMessage` helper for known `InputPeer` values
- Pure MTProto user account phone-code login with `auth.sendCode` and `auth.signIn`
- `updates.getState` helper for authorized bot/user sessions
- Pure MTProto user account phone-code login with `auth.sendCode` and `auth.signIn`
- `updates.getState` helper for authorized bot/user sessions
- No external Go dependencies
- GitHub Actions CI
- Makefile
- Tests and examples

Not complete yet:

- MTProto updates loop and `/start` receive over MTProto
- Automatic peer database and access-hash discovery
- 2FA/SRP password login
- Generated full raw Telegram API
- MTProto update state and gap recovery
- Media upload/download over MTProto
- Full Kurigram-style high-level client parity

## Installation

```bash
go get github.com/growxupdate/MtProGo@main
```

## Real bot example

```bash
go run github.com/growxupdate/MtProGo/examples/realbot@main
```

It asks for:

```text
Enter API ID:
Enter API Hash:
Enter Bot Token:
```

Then send `/start` to your bot. The bot should reply:

```text
Hello from MtProGo V9 🚀
```

This runtime uses Telegram Bot API internally, implemented with the Go standard library.

## Minimal bot code

```go
package main

import (
    "context"

    mtprogo "github.com/growxupdate/MtProGo"
    "github.com/growxupdate/MtProGo/filters"
    "github.com/growxupdate/MtProGo/updates"
)

func main() {
    bot := mtprogo.Must(mtprogo.NewBot(mtprogo.BotConfig{
        APIID:   12345,
        APIHash: "your_api_hash",
        Token:   "your_bot_token",
    }))

    bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
        return m.Reply(ctx, "Hello from MtProGo V9 🚀")
    })

    mtprogo.Must0(bot.Run(context.Background()))
}
```

## Pure MTProto probe

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_probe@main
```

Press Enter to try default Telegram DCs. A successful result prints `pq`, factors `p` and `q`, and RSA fingerprints.

## Pure MTProto auth key generation

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_auth@main
```

Press Enter to try default Telegram DCs. A successful result prints:

```text
Pure MTProto auth key OK
Address: ...
DC ID: ...
RSA fingerprint: ...
Auth key bytes: 256
Auth key ID: ...
Server salt: ...
Server time offset: ...
```

## Encrypted MTProto help.getConfig

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_config@main
```

Press Enter to send direct `help.getConfig`, or enter your API ID to wrap the request with `invokeWithLayer/initConnection`.

A successful result prints:

```text
Encrypted MTProto help.getConfig OK
Address: ...
DC ID: ...
Auth key ID: ...
Server salt: ...
Session ID: ...
Request msg ID: ...
Response msg ID: ...
Result constructor: 0x...
Gzip packed: ...
Result bytes: ...
```

## Pure MTProto bot authorization

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_bot_login@main
```

It asks for:

```text
Enter API ID:
Enter API Hash:
Enter Bot Token:
```

A successful result prints:

```text
Pure MTProto bot authorization OK
Address: ...
DC ID: ...
Auth key ID: ...
Server salt: ...
Session ID: ...
Result constructor: 0x2ea2c0d4 auth.authorization
Result bytes: ...
```

## Pure MTProto account login

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_account_login@main
```

It asks for API ID, API hash, phone number, and the login code sent by Telegram.

A successful login prints `auth.signIn OK` and then calls `updates.getState` to verify the authorized session.

If Telegram returns `SESSION_PASSWORD_NEEDED`, that account has 2FA enabled. V9 supports phone-code login and SRP 2FA password login.

## Raw MTProto messages.sendMessage

```bash
go run github.com/growxupdate/MtProGo/examples/mtproto_send_message@main
```

This example first logs the bot in with `auth.importBotAuthorization`, then calls `messages.sendMessage`.

Important: `inputPeerUser` and `inputPeerChannel` require the Telegram `access_hash`. In a full client, access hashes are learned from MTProto updates, dialogs, contacts, or resolved peers. That automatic peer database is planned for the next milestone.

## Verify dependencies

```bash
go list -m all
```

Expected output:

```text
github.com/growxupdate/MtProGo
```

Also audit Telegram client imports:

```bash
grep -R -n -E "github.com/gotd|tdlib|tgbotapi|telethon|pyrogram|kurigram" . \
  --include="*.go" \
  --include="go.mod" \
  --include="go.sum" \
  --exclude-dir=".git"
```

Expected output: empty.

## Development

```bash
go mod tidy
go test ./...
go vet ./...
make ci
```

## Roadmap

V9 target:

- SRP 2FA password login
- MTProto updates loop and peer cache
- `/start` receive and reply over MTProto
- Basic peer cache and access-hash capture

V9 target:

- Phone login, 2FA, and user accounts
- Updates state tracking
- High-level Kurigram-style helpers

V10 target:

- File upload/download over MTProto
- Generated raw API expansion
