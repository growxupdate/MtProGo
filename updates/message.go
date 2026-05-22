package updates

import (
	"context"
	"time"
)

const (
	// ChatPrivate is a one-to-one user/bot chat.
	ChatPrivate = "private"
	// ChatGroup is a legacy basic group chat.
	ChatGroup = "group"
	// ChatSupergroup is a Telegram supergroup represented by MTProto PeerChannel.
	ChatSupergroup = "supergroup"
	// ChatChannel is a broadcast channel represented by MTProto PeerChannel.
	ChatChannel = "channel"
)

const channelDialogIDOffset int64 = 1000000000000

// NormalizeChatID converts raw MTProto peer IDs into Telegram/Bot-API-style
// dialog IDs used by high-level handlers.
//
// Private users stay positive: 12345
// Basic groups become negative: -12345
// Supergroups/channels become -100-prefixed: -10012345
func NormalizeChatID(rawID int64, chatType string) int64 {
	if rawID == 0 {
		return 0
	}
	switch chatType {
	case ChatGroup:
		if rawID < 0 {
			return rawID
		}
		return -rawID
	case ChatSupergroup, ChatChannel:
		if rawID <= -channelDialogIDOffset {
			return rawID
		}
		if rawID < 0 {
			return rawID
		}
		return -channelDialogIDOffset - rawID
	default:
		return rawID
	}
}

// RawChatID converts a high-level ChatID back to the raw MTProto peer id.
func RawChatID(chatID int64) int64 {
	if chatID <= -channelDialogIDOffset {
		return -chatID - channelDialogIDOffset
	}
	if chatID < 0 {
		return -chatID
	}
	return chatID
}

// LooksLikeChannelChatID reports whether chatID uses the -100... supergroup/channel form.
func LooksLikeChannelChatID(chatID int64) bool {
	return chatID <= -channelDialogIDOffset
}

// ReplySender can send replies for a Message.
type ReplySender interface {
	Reply(ctx context.Context, chatID int64, replyToMessageID int, text string) error
}

// MessageEditor can edit text messages for a Message.
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID int64, messageID int, text string) error
}

// MessageDeleter can delete messages for a Message.
type MessageDeleter interface {
	DeleteMessages(ctx context.Context, chatID int64, messageIDs ...int) error
}

// Message is a normalized Telegram message used by handlers.
type Message struct {
	ID int
	// ChatID is the high-level Telegram dialog ID. Private chats are positive,
	// basic groups are negative, and supergroups/channels use the -100... form.
	ChatID int64
	// RawChatID is the raw MTProto peer id before high-level normalization.
	RawChatID int64
	FromID    int64
	ChatType  string
	Text      string
	Date      time.Time
	Raw       any

	ReplySender ReplySender
	Editor      MessageEditor
	Deleter     MessageDeleter
}

// Reply replies to this message using the configured ReplySender.
func (m *Message) Reply(ctx context.Context, text string) error {
	if m.ReplySender == nil {
		return ErrNoReplySender
	}
	return m.ReplySender.Reply(ctx, m.ChatID, m.ID, text)
}

// Edit edits this message using the configured MessageEditor.
func (m *Message) Edit(ctx context.Context, text string) error {
	if m.Editor == nil {
		return ErrNoReplySender
	}
	return m.Editor.EditMessage(ctx, m.ChatID, m.ID, text)
}

// Delete deletes this message using the configured MessageDeleter.
func (m *Message) Delete(ctx context.Context) error {
	if m.Deleter == nil {
		return ErrNoReplySender
	}
	return m.Deleter.DeleteMessages(ctx, m.ChatID, m.ID)
}
