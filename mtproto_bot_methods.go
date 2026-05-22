package mtprogo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

// ClientMessageHandler receives the MTProto bot client together with the message.
type ClientMessageHandler func(context.Context, *MTProtoBot, *updates.Message) error

// OnMessageClient registers a Kurigram-style handler that receives the client and message.
func (b *MTProtoBot) OnMessageClient(filter filters.Filter, handler ClientMessageHandler) {
	if handler == nil {
		return
	}
	b.OnMessage(filter, func(ctx context.Context, m *updates.Message) error {
		return handler(ctx, b, m)
	})
}

// GetMe returns the current bot/account using users.getUsers(inputUserSelf).
func (b *MTProtoBot) GetMe(ctx context.Context) (*User, error) {
	if b == nil || b.client == nil {
		return nil, errors.New("mtproto bot is not logged in")
	}
	var user *User
	err := b.withFloodWait(ctx, func(callCtx context.Context) error {
		u, _, err := b.client.GetMe(callCtx)
		if err == nil {
			user = u
		}
		return err
	})
	return user, err
}

// ResolveUsername resolves @username and stores the returned peer in the peer cache.
func (b *MTProtoBot) ResolveUsername(ctx context.Context, username string) (PeerRef, error) {
	if b == nil || b.client == nil {
		return PeerRef{}, errors.New("mtproto bot is not logged in")
	}
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return PeerRef{}, errors.New("username is required")
	}
	var peer PeerRef
	err := b.withFloodWait(ctx, func(callCtx context.Context) error {
		p, _, err := b.client.ResolveUsername(callCtx, username)
		if err == nil {
			peer = p
			b.rememberPeers([]PeerRef{p})
		}
		return err
	})
	return peer, err
}

// GetChat returns a cached chat/channel/user peer by high-level chat ID.
func (b *MTProtoBot) GetChat(_ context.Context, chatID int64) (PeerRef, error) {
	peer, ok := b.getPeer(chatID)
	if !ok {
		return PeerRef{}, fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	return peer, nil
}

// GetUser returns a cached user peer by user ID.
func (b *MTProtoBot) GetUser(_ context.Context, userID int64) (PeerRef, error) {
	peer, ok := b.getPeer(userID)
	if !ok || peer.Kind != "user" {
		return PeerRef{}, fmt.Errorf("mtproto: user %d is not cached yet", userID)
	}
	return peer, nil
}

// ForwardMessage forwards one message from one cached chat to another.
func (b *MTProtoBot) ForwardMessage(ctx context.Context, fromChatID, toChatID int64, messageID int, opts ForwardOptions) error {
	return b.ForwardMessages(ctx, fromChatID, toChatID, []int{messageID}, opts)
}

// ForwardMessages forwards messages between cached peers.
func (b *MTProtoBot) ForwardMessages(ctx context.Context, fromChatID, toChatID int64, messageIDs []int, opts ForwardOptions) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	from, ok := b.getPeer(fromChatID)
	if !ok {
		return fmt.Errorf("mtproto: source peer %d is not cached yet", fromChatID)
	}
	to, ok := b.getPeer(toChatID)
	if !ok {
		return fmt.Errorf("mtproto: target peer %d is not cached yet", toChatID)
	}
	ids := make([]int32, 0, len(messageIDs))
	for _, id := range messageIDs {
		if id > 0 {
			ids = append(ids, int32(id))
		}
	}
	if len(ids) == 0 {
		return errors.New("at least one message id is required")
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesForwardMessagesToPeer(callCtx, from, to, ids, toMTProtoForwardOptions(opts))
		return err
	})
}

// GetHistory fetches raw messages.getHistory bytes for a cached peer.
func (b *MTProtoBot) GetHistory(ctx context.Context, chatID int64, limit int32) ([]byte, error) {
	if b == nil || b.client == nil {
		return nil, errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	input, ok := peer.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d (%s) is missing access data", peer.ID, peer.Kind)
	}
	var body []byte
	err := b.withFloodWait(ctx, func(callCtx context.Context) error {
		res, err := b.client.MessagesGetHistory(callCtx, input, limit)
		if err == nil {
			body = append([]byte(nil), res.Body...)
		}
		return err
	})
	return body, err
}
