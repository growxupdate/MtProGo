# MtProGo

MtProGo is a pure Go Telegram client project focused on a clean API, low memory use, and a long-term MTProto implementation.

This repository intentionally does **not** import or wrap TDLib, gotd, tgbotapi, Telethon, Pyrogram, Kurigram, or any other Telegram client library.

## Current V11 status

Working now:

- Real Telegram bot runtime using the Go standard library Bot API backend
- Real `/start` bot example with replies
- Pure Go MTProto `req_pq_multi` probe
- Pure Go MTProto auth key generation
- Pure Go encrypted MTProto `help.getConfig`
- Pure MTProto bot authorization with DC migration handling
- Pure MTProto account phone-code login
- Pure MTProto SRP 2FA password login
- Configurable update handling
- Configurable bounded message cache
- Configurable bounded peer cache
- No external Go dependencies

Still in progress:

- Full generated TL API
- Full MTProto updates dispatcher
- High-level Kurigram/Pyrogram-style helpers for every Telegram method
- Media upload/download helpers

## Install

```bash
go get github.com/growxupdate/MtProGo@main
```

## Basic bot example

```go
package main

import (
    "context"

    mtprogo "github.com/growxupdate/MtProGo"
    "github.com/growxupdate/MtProGo/filters"
    "github.com/growxupdate/MtProGo/updates"
)

func main() {
    bot := mtprogo.Must(mtprogo.NewBotWithOptions(
        mtprogo.BotConfig{Token: "BOT_TOKEN"},
        mtprogo.WithUpdates(true),
        mtprogo.WithMessageCacheSize(1000),
        mtprogo.WithPeerCacheSize(1000),
    ))

    bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
        return m.Reply(ctx, "Hello from MtProGo V11 🚀")
    })

    mtprogo.Must0(bot.Run(context.Background()))
}
```

## Memory/cache controls

MtProGo V11 lets users decide how much data should stay in RAM.

The default message cache policy is **matched-only**. That means MtProGo does **not** cache every random update. If your bot registers only `/start`, `/ping`, `/help`, and `/queue`, then only messages matching registered handlers are cached by default. This is safer for very large bots because irrelevant messages are processed and dropped instead of being retained in RAM.

```go
client := mtprogo.Must(mtprogo.New(
    12345,
    "api_hash",
    mtprogo.WithUpdates(true),
    mtprogo.WithMessageCacheSize(1000), // keep last 1000 messages
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


Selective command cache:

```go
bot := mtprogo.Must(mtprogo.NewBotWithOptions(
    mtprogo.BotConfig{Token: token},
    mtprogo.WithUpdates(true),
    mtprogo.WithMessageCacheSize(1000),
    mtprogo.WithPeerCacheSize(1000),
    mtprogo.WithMessageCacheCommands("start", "ping", "help", "queue"),
))
```

Cache policy options:

```go
mtprogo.WithMessageCacheMode(mtprogo.CacheMatchedMessages)  // default: cache only handled messages
mtprogo.WithMessageCacheMode(mtprogo.CacheAllMessages)      // cache every incoming message
mtprogo.WithMessageCacheMode(mtprogo.CacheNone)             // cache nothing
mtprogo.WithMessageCacheFilter(customFilter)                // cache only messages accepted by your filter
```

When the cache limit is full, MtProGo drops the oldest cached message and keeps the newest one. With `WithMessageCacheSize(1000)`, message 1001 evicts message 1.

Disable message cache:

```go
client := mtprogo.Must(mtprogo.New(
    12345,
    "api_hash",
    mtprogo.WithMessageCacheSize(0),
))
```

Disable handler dispatching while still allowing cache population:

```go
client := mtprogo.Must(mtprogo.New(
    12345,
    "api_hash",
    mtprogo.WithUpdates(false),
))
```

Read cache stats:

```go
snapshot := client.CacheSnapshot()
println(snapshot.MessageCount, snapshot.MessageLimit)
println(snapshot.PeerCount, snapshot.PeerLimit)
```

## Examples

Run a real bot:

```bash
go run ./examples/realbot
```

Run a real bot with cache options:

```bash
go run ./examples/realbot_cache
```

Run local cache option demo:

```bash
go run ./examples/cache_options
```

Run selective cache demo with 10,000 simulated updates:

```bash
go run ./examples/selective_cache
```

Run real bot with selective command cache:

```bash
go run ./examples/realbot_selective_cache
```

Run pure MTProto probe:

```bash
go run ./examples/mtproto_probe
```

Generate a pure MTProto auth key:

```bash
go run ./examples/mtproto_auth
```

Call encrypted MTProto help.getConfig:

```bash
go run ./examples/mtproto_config
```

Login a bot over pure MTProto:

```bash
go run ./examples/mtproto_bot_login
```

Login an account over pure MTProto:

```bash
go run ./examples/mtproto_account_login
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
