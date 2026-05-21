package updates

import (
	"context"
	"time"
)

// ReplySender can send replies for a Message.
type ReplySender interface {
	Reply(ctx context.Context, chatID int64, replyToMessageID int, text string) error
}

// Message is a normalized Telegram message used by handlers.
type Message struct {
	ID     int
	ChatID int64
	FromID int64
	Text   string
	Date   time.Time
	Raw    any

	ReplySender ReplySender
}

// Reply replies to this message using the configured ReplySender.
func (m *Message) Reply(ctx context.Context, text string) error {
	if m.ReplySender == nil {
		return ErrNoReplySender
	}
	return m.ReplySender.Reply(ctx, m.ChatID, m.ID, text)
}
