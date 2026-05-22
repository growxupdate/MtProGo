# MtProGo

MtProGo is a pure Go Telegram client project focused on a clean API, low memory use, and a long-term MTProto implementation.

This repository intentionally does **not** import or wrap TDLib, gotd, tgbotapi, Telethon, Pyrogram, Kurigram, or any other Telegram client library.

## Current V16 status

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
- Experimental pure MTProto private/basic-group/supergroup text update parsing and reply helpers
- Parses `updateNewMessage` and `updateNewChannelMessage` from `updates.getDifference`
- Persistent MTProto sessions for bots and user accounts
- File sessions, encrypted file sessions, and copy-paste string sessions
- Configurable update handling, bounded message cache, and bounded peer cache
- V16 reliability hardening: persisted update state, persisted peer cache, bad-salt/bad-msg retry, reconnect backoff, optional FLOOD_WAIT auto sleep
- No external Go dependencies

Still in progress:

- Full generated TL API
- Full generated MTProto updates parser for every update type
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
    mtprogo.WithAutoReconnect(true),
    mtprogo.WithReconnectBackoff(time.Second, 30*time.Second),
    mtprogo.WithAutoFloodWait(true, 60*time.Second),
))

bot.OnMessage(filters.Command("start"), func(ctx context.Context, m *updates.Message) error {
    return m.Reply(ctx, "Hello from pure MTProto")
})

bot.OnMessage(filters.And(filters.Command("ping"), filters.Group), func(ctx context.Context, m *updates.Message) error {
    return m.Reply(ctx, "group pong")
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


## MTProto chat helpers and production reliability in V16

V16 keeps normalized chat types and adds Bot-API-style chat IDs for high-level handlers:

```go
bot.OnMessage(filters.Command("chatid"), func(ctx context.Context, m *updates.Message) error {
    return bot.SendMessage(ctx, m.ChatID, fmt.Sprintf("chat_id=%d type=%s", m.ChatID, m.ChatType))
})

bot.OnMessage(filters.And(filters.Command("ping"), filters.Private), privateHandler)
bot.OnMessage(filters.And(filters.Command("ping"), filters.Group), groupHandler)
bot.OnMessage(filters.And(filters.Command("ping"), filters.Channel), channelHandler)
```

Available message helpers:

```go
m.Reply(ctx, "text")
m.Edit(ctx, "edited text")
m.Delete(ctx)
bot.SendMessage(ctx, chatID, "text")
bot.EditMessage(ctx, chatID, messageID, "edited text")
bot.DeleteMessages(ctx, chatID, messageID)
```

The MTProto update parser now recognizes `PeerUser`, `PeerChat`, and `PeerChannel`. Channel/supergroup sending requires the peer access hash to be present in the cached updates payload.

High-level `Message.ChatID` values now match Telegram/Bot-API style IDs:

- Private: `123456789`
- Basic group: `-123456789`
- Supergroup/channel: `-100123456789`

The raw positive MTProto peer ID remains available as `Message.RawChatID`. You can pass the public `Message.ChatID` back to `Reply`, `SendMessage`, and `EditMessage`; MtProGo resolves it to the cached raw peer internally.


## V16 production reliability controls

V16 stores the current `pts/qts/date/seq` update state and the lightweight peer cache inside the bot session file. On restart, MtProGo can resume from the saved update state instead of always starting from a fresh `updates.getState`. It also keeps saved access hashes for peers already seen in updates, which avoids losing group/supergroup send ability after restart.

```go
bot := mtprogo.Must(mtprogo.NewMTProtoBot(
    mtprogo.MTProtoBotConfig{APIID: apiID, APIHash: apiHash, Token: botToken},
    mtprogo.WithSessionFile("bot.session"),
    mtprogo.WithAutoReconnect(true),
    mtprogo.WithReconnectBackoff(time.Second, 30*time.Second),
    mtprogo.WithAutoFloodWait(true, 60*time.Second),
    mtprogo.WithDebug(false),
))
```

Reliability pieces added in V16:

- `bad_server_salt` updates the server salt and automatically retries the request
- retriable `bad_msg_notification` codes retry the request
- `FLOOD_WAIT_X` is exposed as a typed helper and can be auto-slept up to your limit
- temporary network errors reconnect with exponential backoff
- session save happens during run and on graceful shutdown
- update state and peer cache are persisted with the MTProto auth key

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
    return m.Reply(ctx, "Hello from MtProGo V16 🚀")
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
go run ./examples/mtproto_chat_helpers
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


## V14.2 group/supergroup access-hash fix

MtProGo now scans the current layer `channel#fe685355` constructor from update chat vectors, while keeping legacy constructor support. This fixes supergroup replies that were received correctly but failed with missing channel access data.

## V15 chat ID normalization

MtProGo now exposes public chat IDs the same way Telegram/Bot API users expect:

- Private: `123456789`
- Basic group: `-123456789`
- Supergroup/channel: `-100123456789`

`Message.RawChatID` keeps the original positive MTProto peer id for debugging and low-level work. High-level methods such as `Message.Reply`, `bot.SendMessage`, and `bot.EditMessage` accept the normalized public `ChatID` and resolve it back to the cached MTProto peer internally.
