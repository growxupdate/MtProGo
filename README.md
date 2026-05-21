# MtProGo

MtProGo is a pure Go Telegram client project focused on a clean API, low memory use, and a long-term MTProto implementation.

This repository intentionally does **not** import or wrap TDLib, gotd, tgbotapi, Telethon, Pyrogram, Kurigram, or any other Telegram client library.

## Current V13 status

Working now:

- Real Telegram bot runtime using the Go standard library Bot API backend
- Real `/start` bot example with replies
- Pure Go MTProto `req_pq_multi` probe
- Pure Go MTProto auth key generation
- Pure Go encrypted MTProto `help.getConfig`
- Pure MTProto bot authorization with DC migration handling
- Pure MTProto account phone-code login
- Pure MTProto SRP 2FA password login
- Experimental pure MTProto bot updates loop using `updates.getDifference`
- Experimental pure MTProto private `/start`, `/ping`, `/help` receive/reply example
- Persistent MTProto sessions for bots and user accounts
- File sessions, encrypted file sessions, and copy-paste string sessions
- Configurable update handling, bounded message cache, and bounded peer cache
- No external Go dependencies

Still in progress:

- Full generated TL API
- Full generated MTProto updates parser for every update type
- Group/channel peer parsing for MTProto updates
- High-level Kurigram/Pyrogram-style helpers for every Telegram method
- Media upload/download helpers

## Install

```bash
go get github.com/growxupdate/MtProGo@main
```

## Pure MTProto bot with persistent session

First run creates `bot.session`. Later runs load it and skip auth-key generation and `auth.importBotAuthorization`, which makes startup faster and lighter.

```bash
go run ./examples/mtproto_bot_updates
```

Minimal code:

```go
bot := mtprogo.Must(mtprogo.NewMTProtoBot(
    mtprogo.MTProtoBotConfig{
        APIID:   apiID,
        APIHash: apiHash,
        Token:   botToken,
    },
    mtprogo.WithSessionFile("bot.session"),
    mtprogo.WithUpdates(true),
    mtprogo.WithMessageCacheSize(1000),
    mtprogo.WithPeerCacheSize(1000),
    mtprogo.WithMessageCacheCommands("start", "ping", "help"),
))

bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
    return m.Reply(ctx, "Hello from pure MTProto")
})

mtprogo.Must0(bot.Run(context.Background()))
```

Encrypted session file:

```go
bot := mtprogo.Must(mtprogo.NewMTProtoBot(
    mtprogo.MTProtoBotConfig{APIID: apiID, APIHash: apiHash, Token: botToken},
    mtprogo.WithEncryptedSessionFile("bot.session", "strong-password"),
))
```

String session import:

```go
bot := mtprogo.Must(mtprogo.NewMTProtoBot(
    mtprogo.MTProtoBotConfig{APIID: apiID, APIHash: apiHash, Token: botToken},
    mtprogo.WithStringSession("MPG1:..."),
))
```

Export a string session after login:

```go
encoded, err := bot.ExportStringSession()
```

Clear saved session:

```go
err := bot.ClearSession(context.Background())
```

## User account persistent session

```bash
go run ./examples/mtproto_account_login
```

The first run asks for API ID, API hash, phone number, code, and 2FA password when needed. It saves `account.session` and prints a string session. The next run loads the session and checks `updates.getState` without asking for the phone code again.

## Bot API runtime

The Bot API runtime is still available for simple bots:

```go
bot := mtprogo.Must(mtprogo.NewBotWithOptions(
    mtprogo.BotConfig{Token: "BOT_TOKEN"},
    mtprogo.WithUpdates(true),
    mtprogo.WithMessageCacheSize(1000),
    mtprogo.WithPeerCacheSize(1000),
))

bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
    return m.Reply(ctx, "Hello from MtProGo V13 🚀")
})

mtprogo.Must0(bot.Run(context.Background()))
```

## Memory/cache controls

The default message cache policy is **matched-only**. MtProGo does **not** cache every random update. If your bot registers only `/start`, `/ping`, `/help`, and `/queue`, then only messages matching registered handlers are cached by default. Irrelevant updates are processed and dropped instead of retained in RAM.

```go
client := mtprogo.Must(mtprogo.New(
    12345,
    "api_hash",
    mtprogo.WithUpdates(true),
    mtprogo.WithMessageCacheSize(1000), // keep last 1000 useful messages
    mtprogo.WithPeerCacheSize(1000),    // keep last 1000 peers
    mtprogo.WithUpdateQueueSize(256),
))
```

Low-memory setup:

```go
client := mtprogo.Must(mtprogo.New(
    12345,
    "api_hash",
    mtprogo.WithMessageCacheSize(100),
    mtprogo.WithPeerCacheSize(100),
))
```

Cache policy options:

```go
mtprogo.WithMessageCacheMode(mtprogo.CacheMatchedMessages)  // default: cache only handled messages
mtprogo.WithMessageCacheMode(mtprogo.CacheAllMessages)      // cache every incoming message
mtprogo.WithMessageCacheMode(mtprogo.CacheNone)             // cache nothing
mtprogo.WithMessageCacheFilter(customFilter)                // cache only messages accepted by your filter
mtprogo.WithMessageCacheCommands("start", "ping", "help")  // command-only cache
```

When the cache limit is full, MtProGo drops the oldest cached message and keeps the newest one. With `WithMessageCacheSize(1000)`, message 1001 evicts message 1.

## Examples

```bash
go run ./examples/realbot
go run ./examples/realbot_cache
go run ./examples/cache_options
go run ./examples/selective_cache
go run ./examples/realbot_selective_cache
go run ./examples/mtproto_probe
go run ./examples/mtproto_auth
go run ./examples/mtproto_config
go run ./examples/mtproto_bot_login
go run ./examples/mtproto_bot_updates
go run ./examples/mtproto_account_login
go run ./examples/mtproto_session_tools
```

## CI

```bash
go mod tidy
go test ./...
go vet ./...
make ci
```

## Audit

```bash
go list -m all

grep -R -n -E "github.com/gotd|tdlib|tgbotapi|telethon|pyrogram|kurigram" . \
  --include="*.go" \
  --include="go.mod" \
  --include="go.sum" \
  --exclude-dir=".git"
```

Expected module audit:

```text
github.com/growxupdate/MtProGo
```

Expected external Telegram library audit: empty output.
