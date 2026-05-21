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
	api, err := botapi.New(config.Token)
	if err != nil {
		return nil, err
	}
	return &Bot{
		config:     config,
		api:        api,
		dispatcher: updates.NewDispatcher(),
	}, nil
}

// OnMessage registers a bot message handler.
func (b *Bot) OnMessage(filter filters.Filter, handler updates.MessageHandler) {
	b.dispatcher.OnMessage(filter, handler)
}

// Dispatcher returns the bot dispatcher.
func (b *Bot) Dispatcher() *updates.Dispatcher { return b.dispatcher }

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
	offset := 0
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
	return &updates.Message{
		ID:          message.MessageID,
		ChatID:      message.Chat.ID,
		FromID:      fromID,
		Text:        strings.TrimSpace(message.Text),
		Raw:         message,
		ReplySender: sender,
	}
}
