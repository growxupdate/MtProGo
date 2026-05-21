package mtprogo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/internal/botapi"
	"github.com/growxupdate/MtProGo/updates"
)

// BotConfig configures a real Bot API runtime.
type BotConfig struct {
	APIID       int
	APIHash     string
	Token       string
	Session     string
	PollTimeout time.Duration

	// Updates enables getUpdates polling and handler dispatching. Default: true.
	Updates bool
	// DropPendingUpdates drops old Bot API updates before starting the loop.
	DropPendingUpdates bool
	// MessageCacheSize controls recent message cache size. Default: 1000.
	MessageCacheSize int
	// PeerCacheSize controls cached peer count. Default: 1000.
	PeerCacheSize int
	// UpdateQueueSize reserves future async update queue capacity. Default: 256.
	UpdateQueueSize int
}

// Bot is a real Telegram bot runtime built with the Go standard library.
type Bot struct {
	config     BotConfig
	api        *botapi.Client
	dispatcher *updates.Dispatcher
	me         botapi.User
}

// NewBot creates a real Telegram bot runtime.
func NewBot(config BotConfig) (*Bot, error) {
	if config.Token == "" {
		return nil, errors.New("bot token is required")
	}
	if config.PollTimeout <= 0 {
		config.PollTimeout = 30 * time.Second
	}
	if config.MessageCacheSize == 0 {
		config.MessageCacheSize = updates.DefaultMessageCacheSize
	}
	if config.PeerCacheSize == 0 {
		config.PeerCacheSize = updates.DefaultPeerCacheSize
	}
	if config.UpdateQueueSize == 0 {
		config.UpdateQueueSize = 256
	}
	// Updates defaults to true. Since bool cannot distinguish omitted from false,
	// use NewBotWithOptions when explicit false is needed without struct literals.
	config.Updates = true
	api, err := botapi.New(config.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		config: config,
		api:    api,
		dispatcher: updates.NewDispatcher(
			updates.WithUpdates(config.Updates),
			updates.WithMessageCacheSize(config.MessageCacheSize),
			updates.WithPeerCacheSize(config.PeerCacheSize),
			updates.WithUpdateQueueSize(config.UpdateQueueSize),
		),
	}, nil
}

// NewBotWithOptions creates a bot and applies memory/update options reliably.
func NewBotWithOptions(config BotConfig, opts ...Option) (*Bot, error) {
	cfg := Config{
		APIID:            config.APIID,
		APIHash:          config.APIHash,
		Session:          MemorySession(),
		Updates:          true,
		MessageCacheSize: updates.DefaultMessageCacheSize,
		PeerCacheSize:    updates.DefaultPeerCacheSize,
		UpdateQueueSize:  256,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	config.Updates = cfg.Updates
	config.MessageCacheSize = cfg.MessageCacheSize
	config.PeerCacheSize = cfg.PeerCacheSize
	config.UpdateQueueSize = cfg.UpdateQueueSize
	return newBotExact(config)
}

func newBotExact(config BotConfig) (*Bot, error) {
	if config.Token == "" {
		return nil, errors.New("bot token is required")
	}
	if config.PollTimeout <= 0 {
		config.PollTimeout = 30 * time.Second
	}
	if config.MessageCacheSize < 0 {
		config.MessageCacheSize = 0
	}
	if config.PeerCacheSize < 0 {
		config.PeerCacheSize = 0
	}
	if config.UpdateQueueSize < 0 {
		config.UpdateQueueSize = 0
	}
	api, err := botapi.New(config.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		config: config,
		api:    api,
		dispatcher: updates.NewDispatcher(
			updates.WithUpdates(config.Updates),
			updates.WithMessageCacheSize(config.MessageCacheSize),
			updates.WithPeerCacheSize(config.PeerCacheSize),
			updates.WithUpdateQueueSize(config.UpdateQueueSize),
		),
	}, nil
}

// OnMessage registers a bot message handler.
func (b *Bot) OnMessage(filter filters.Filter, handler updates.MessageHandler) {
	b.dispatcher.OnMessage(filter, handler)
}

// Dispatcher returns the bot dispatcher.
func (b *Bot) Dispatcher() *updates.Dispatcher { return b.dispatcher }

// CacheSnapshot returns current message/peer cache sizes.
func (b *Bot) CacheSnapshot() updates.CacheSnapshot { return b.dispatcher.CacheSnapshot() }

// Me returns the bot user after Run starts or Login is called.
func (b *Bot) Me() botapi.User { return b.me }

// Login validates the token and loads bot identity.
func (b *Bot) Login(ctx context.Context) error {
	me, err := b.api.GetMe(ctx)
	if err != nil {
		return err
	}
	b.me = me
	return nil
}

// Run starts long polling and dispatches real Telegram messages.
func (b *Bot) Run(ctx context.Context) error {
	if b.me.ID == 0 {
		if err := b.Login(ctx); err != nil {
			return err
		}
	}

	fmt.Printf("MtProGo bot connected as @%s\n", b.me.Username)
	if !b.config.Updates {
		fmt.Println("updates disabled; bot is logged in and waiting for context cancellation")
		<-ctx.Done()
		return ctx.Err()
	}

	offset := 0
	if b.config.DropPendingUpdates {
		old, err := b.api.GetUpdates(ctx, -1, 0)
		if err == nil && len(old) > 0 {
			offset = old[len(old)-1].UpdateID + 1
		}
	}
	timeoutSeconds := int(b.config.PollTimeout / time.Second)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batch, err := b.api.GetUpdates(ctx, offset, timeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			fmt.Println("getUpdates error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range batch {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			if update.Message == nil {
				continue
			}
			msg := botAPIMessageToUpdate(update.Message, b)
			if err := b.dispatcher.DispatchMessage(ctx, msg); err != nil {
				fmt.Println("handler error:", err)
			}
		}
	}
}

// Reply implements updates.ReplySender.
func (b *Bot) Reply(ctx context.Context, chatID int64, replyToMessageID int, text string) error {
	return b.api.SendMessage(ctx, chatID, text, replyToMessageID)
}

// SendMessage sends a message without a reply target.
func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) error {
	return b.api.SendMessage(ctx, chatID, text, 0)
}

func botAPIMessageToUpdate(message *botapi.Message, sender updates.ReplySender) *updates.Message {
	var fromID int64
	if message.From != nil {
		fromID = message.From.ID
	}
	msg := &updates.Message{
		ID:          message.MessageID,
		ChatID:      message.Chat.ID,
		FromID:      fromID,
		Text:        strings.TrimSpace(message.Text),
		Raw:         message,
		ReplySender: sender,
	}
	if fromID != 0 && message.Chat.ID == fromID {
		msg.Date = time.Now()
	}
	return msg
}
