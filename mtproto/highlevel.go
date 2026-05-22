package mtproto

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	constructorUsersGetUsers        = 0x0d91a548
	constructorContactsResolveUser  = 0x725afbbc
	constructorContactsResolvedPeer = 0x7f077ad9
	constructorInputUserSelf        = 0xf7c1b13f
	constructorInputUser            = 0xf21158c6
	constructorInputReplyToMessage  = 0x869fbe10
	constructorMessagesForward      = 0x978928ca
	constructorMessagesGetHistory   = 0x4423e6c5
	constructorChannelsDelete       = 0x84c1fd4e
	constructorInputChannel         = 0xf35aec28
)

// User is a small parsed Telegram user/bot profile returned by high-level helpers.
type User struct {
	ID         int64
	AccessHash int64
	Username   string
	FirstName  string
	LastName   string
	Phone      string
	Bot        bool
	Self       bool
}

// DisplayName returns a compact human-readable name.
func (u User) DisplayName() string {
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName + " " + u.LastName))
	if name != "" {
		return name
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	if u.ID != 0 {
		return fmt.Sprintf("%d", u.ID)
	}
	return ""
}

// SendOptions controls optional flags for messages.sendMessage.
type SendOptions struct {
	ReplyToMessageID int
	Silent           bool
	NoWebpage        bool
	Background       bool
}

// EditOptions controls optional flags for messages.editMessage.
type EditOptions struct {
	NoWebpage bool
}

// ForwardOptions controls optional flags for messages.forwardMessages.
type ForwardOptions struct {
	Silent            bool
	DropAuthor        bool
	DropMediaCaptions bool
}

// InputUser is a raw MTProto InputUser helper.
type InputUser struct {
	kind       string
	id         int64
	accessHash int64
}

// InputUserSelf returns inputUserSelf.
func InputUserSelf() InputUser { return InputUser{kind: "self"} }

// InputUserID returns inputUser with access_hash.
func InputUserID(userID, accessHash int64) InputUser {
	return InputUser{kind: "user", id: userID, accessHash: accessHash}
}

func (u InputUser) encode(b *tlBuffer) error {
	switch u.kind {
	case "self":
		b.putInt(constructorInputUserSelf)
	case "user":
		if u.id == 0 || u.accessHash == 0 {
			return errors.New("inputUser requires user id and access hash")
		}
		b.putInt(constructorInputUser)
		b.putLong(uint64(u.id))
		b.putLong(uint64(u.accessHash))
	default:
		return errors.New("unknown input user kind")
	}
	return nil
}

// GetMe calls users.getUsers([inputUserSelf]) and returns the current account/bot.
func (c *EncryptedClient) GetMe(ctx context.Context) (*User, *InvokeResult, error) {
	result, err := c.Invoke(ctx, makeUsersGetUsersQuery(InputUserSelf()))
	if err != nil {
		return nil, nil, err
	}
	users := ScanUsers(result.Body)
	if len(users) == 0 {
		return nil, result, errors.New("mtproto: users.getUsers returned no parsed users")
	}
	result.Message = "users.getUsers(inputUserSelf) succeeded over encrypted MTProto"
	return &users[0], result, nil
}

// ResolveUsername resolves a public @username and returns a cached PeerRef candidate.
func (c *EncryptedClient) ResolveUsername(ctx context.Context, username string) (PeerRef, *InvokeResult, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return PeerRef{}, nil, errors.New("username is required")
	}
	result, err := c.Invoke(ctx, makeContactsResolveUsernameQuery(username))
	if err != nil {
		return PeerRef{}, nil, err
	}
	peer, ok := parseResolvedPeer(result.Body)
	if !ok {
		return PeerRef{}, result, errors.New("mtproto: unable to parse resolved peer")
	}
	if peer.Username == "" {
		peer.Username = username
	}
	result.Message = "contacts.resolveUsername succeeded over encrypted MTProto"
	return peer, result, nil
}

// MessagesSendMessageWithOptions sends text with reply/silent/no_webpage flags.
func (c *EncryptedClient) MessagesSendMessageWithOptions(ctx context.Context, peer InputPeer, text string, opts SendOptions) (*InvokeResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text is required")
	}
	query, err := makeMessagesSendMessageQueryWithOptions(peer, text, opts)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "messages.sendMessage succeeded over encrypted MTProto"
	return result, nil
}

// MessagesSendMessageToPeerWithOptions sends text to a parsed peer reference.
func (c *EncryptedClient) MessagesSendMessageToPeerWithOptions(ctx context.Context, peer PeerRef, text string, opts SendOptions) (*InvokeResult, error) {
	input, ok := peer.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d (%s) is missing access data", peer.ID, peer.Kind)
	}
	return c.MessagesSendMessageWithOptions(ctx, input, text, opts)
}

// MessagesEditMessageWithOptions edits text with optional no_webpage.
func (c *EncryptedClient) MessagesEditMessageWithOptions(ctx context.Context, peer InputPeer, messageID int32, text string, opts EditOptions) (*InvokeResult, error) {
	if messageID <= 0 {
		return nil, errors.New("message id is required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text is required")
	}
	query, err := makeMessagesEditMessageQueryWithOptions(peer, messageID, text, opts)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "messages.editMessage succeeded over encrypted MTProto"
	return result, nil
}

// MessagesEditMessageToPeerWithOptions edits a message for a parsed peer reference.
func (c *EncryptedClient) MessagesEditMessageToPeerWithOptions(ctx context.Context, peer PeerRef, messageID int32, text string, opts EditOptions) (*InvokeResult, error) {
	input, ok := peer.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: peer %d (%s) is missing access data", peer.ID, peer.Kind)
	}
	return c.MessagesEditMessageWithOptions(ctx, input, messageID, text, opts)
}

// MessagesForwardMessages forwards message IDs from one cached peer to another.
func (c *EncryptedClient) MessagesForwardMessages(ctx context.Context, from, to InputPeer, ids []int32, opts ForwardOptions) (*InvokeResult, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one message id is required")
	}
	query, err := makeMessagesForwardMessagesQuery(from, to, ids, opts)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "messages.forwardMessages succeeded over encrypted MTProto"
	return result, nil
}

// MessagesForwardMessagesToPeer forwards using parsed PeerRef values.
func (c *EncryptedClient) MessagesForwardMessagesToPeer(ctx context.Context, from, to PeerRef, ids []int32, opts ForwardOptions) (*InvokeResult, error) {
	fromInput, ok := from.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: from peer %d (%s) is missing access data", from.ID, from.Kind)
	}
	toInput, ok := to.InputPeer()
	if !ok {
		return nil, fmt.Errorf("mtproto: to peer %d (%s) is missing access data", to.ID, to.Kind)
	}
	return c.MessagesForwardMessages(ctx, fromInput, toInput, ids, opts)
}

// ChannelsDeleteMessages deletes messages from a channel/supergroup.
func (c *EncryptedClient) ChannelsDeleteMessages(ctx context.Context, channel PeerRef, ids ...int32) (*InvokeResult, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one message id is required")
	}
	query, err := makeChannelsDeleteMessagesQuery(channel, ids...)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "channels.deleteMessages succeeded over encrypted MTProto"
	return result, nil
}

// MessagesGetHistory fetches raw history result bytes for a cached peer.
func (c *EncryptedClient) MessagesGetHistory(ctx context.Context, peer InputPeer, limit int32) (*InvokeResult, error) {
	if limit <= 0 {
		limit = 20
	}
	query, err := makeMessagesGetHistoryQuery(peer, limit)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "messages.getHistory succeeded over encrypted MTProto"
	return result, nil
}

func makeUsersGetUsersQuery(users ...InputUser) []byte {
	var b tlBuffer
	b.putInt(constructorUsersGetUsers)
	b.putInt(constructorVector)
	b.putInt(uint32(len(users)))
	for _, user := range users {
		_ = user.encode(&b)
	}
	return b.bytes()
}

func makeContactsResolveUsernameQuery(username string) []byte {
	var b tlBuffer
	b.putInt(constructorContactsResolveUser)
	b.putInt(0) // flags: no referer
	b.putString(username)
	return b.bytes()
}

func makeMessagesSendMessageQueryWithOptions(peer InputPeer, text string, opts SendOptions) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesSendMessage)
	var flags uint32
	if opts.ReplyToMessageID > 0 {
		flags |= 1 << 0
	}
	if opts.NoWebpage {
		flags |= 1 << 1
	}
	if opts.Silent {
		flags |= 1 << 5
	}
	if opts.Background {
		flags |= 1 << 6
	}
	b.putInt(flags)
	if err := peer.encode(&b); err != nil {
		return nil, err
	}
	if opts.ReplyToMessageID > 0 {
		b.putInt(constructorInputReplyToMessage)
		b.putInt(0) // InputReplyTo flags
		b.putInt(uint32(int32(opts.ReplyToMessageID)))
	}
	b.putString(text)
	b.putLong(uint64(randomInt64()))
	return b.bytes(), nil
}

func makeMessagesEditMessageQueryWithOptions(peer InputPeer, messageID int32, text string, opts EditOptions) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesEditMessage)
	flags := uint32(1 << 11) // message
	if opts.NoWebpage {
		flags |= 1 << 1
	}
	b.putInt(flags)
	if err := peer.encode(&b); err != nil {
		return nil, err
	}
	b.putInt(uint32(messageID))
	b.putString(text)
	return b.bytes(), nil
}

func makeMessagesForwardMessagesQuery(from, to InputPeer, ids []int32, opts ForwardOptions) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesForward)
	var flags uint32
	if opts.Silent {
		flags |= 1 << 5
	}
	if opts.DropAuthor {
		flags |= 1 << 11
	}
	if opts.DropMediaCaptions {
		flags |= 1 << 12
	}
	b.putInt(flags)
	if err := from.encode(&b); err != nil {
		return nil, err
	}
	b.putInt(constructorVector)
	b.putInt(uint32(len(ids)))
	for _, id := range ids {
		b.putInt(uint32(id))
	}
	b.putInt(constructorVector)
	b.putInt(uint32(len(ids)))
	for range ids {
		b.putLong(uint64(randomInt64()))
	}
	if err := to.encode(&b); err != nil {
		return nil, err
	}
	return b.bytes(), nil
}

func makeChannelsDeleteMessagesQuery(channel PeerRef, ids ...int32) ([]byte, error) {
	if channel.ID == 0 || channel.AccessHash == 0 {
		return nil, fmt.Errorf("mtproto: channel %d is missing access data", channel.ID)
	}
	var b tlBuffer
	b.putInt(constructorChannelsDelete)
	b.putInt(constructorInputChannel)
	b.putLong(uint64(channel.ID))
	b.putLong(uint64(channel.AccessHash))
	b.putInt(constructorVector)
	b.putInt(uint32(len(ids)))
	for _, id := range ids {
		b.putInt(uint32(id))
	}
	return b.bytes(), nil
}

func makeMessagesGetHistoryQuery(peer InputPeer, limit int32) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesGetHistory)
	if err := peer.encode(&b); err != nil {
		return nil, err
	}
	b.putInt(0) // offset_id
	b.putInt(0) // offset_date
	b.putInt(0) // add_offset
	b.putInt(uint32(limit))
	b.putInt(0)  // max_id
	b.putInt(0)  // min_id
	b.putLong(0) // hash
	return b.bytes(), nil
}

func parseResolvedPeer(body []byte) (PeerRef, bool) {
	if len(body) < 16 || binary.LittleEndian.Uint32(body[:4]) != constructorContactsResolvedPeer {
		peers := scanPeerRefs(body)
		if len(peers) == 0 {
			return PeerRef{}, false
		}
		return peers[0], true
	}
	peerKind, peerID, ok := parsePeerAt(body, 4)
	all := scanPeerRefs(body)
	if !ok {
		if len(all) == 0 {
			return PeerRef{}, false
		}
		return all[0], true
	}
	for _, p := range all {
		if p.ID == peerID && (peerKind == "" || p.Kind == peerKind || (peerKind == "channel" && p.Kind == "supergroup")) {
			return p, true
		}
	}
	return PeerRef{ID: peerID, Kind: peerKind}, true
}

func parsePeerAt(body []byte, off int) (string, int64, bool) {
	if off+12 > len(body) {
		return "", 0, false
	}
	constructor := binary.LittleEndian.Uint32(body[off:])
	id := int64(binary.LittleEndian.Uint64(body[off+4:]))
	switch constructor {
	case constructorPeerUser:
		return "user", id, true
	case constructorPeerChat:
		return "chat", id, true
	case constructorPeerChannel:
		return "channel", id, true
	default:
		return "", 0, false
	}
}

// ScanUsers extracts small user summaries from a raw TL response.
func ScanUsers(body []byte) []User {
	var out []User
	seen := map[int64]struct{}{}
	for off := 0; off+20 <= len(body); off += 4 {
		if binary.LittleEndian.Uint32(body[off:]) != constructorUser {
			continue
		}
		user, ok := parseUserAt(body, off)
		if !ok || user.ID == 0 {
			continue
		}
		if _, exists := seen[user.ID]; exists {
			continue
		}
		seen[user.ID] = struct{}{}
		out = append(out, user)
	}
	return out
}

func parseUserAt(body []byte, off int) (User, bool) {
	r := newTLReader(body[off:])
	constructor, err := r.int()
	if err != nil || constructor != constructorUser {
		return User{}, false
	}
	flags, err := r.int()
	if err != nil {
		return User{}, false
	}
	flags2, err := r.int()
	if err != nil {
		return User{}, false
	}
	_ = flags2
	id, err := r.long()
	if err != nil {
		return User{}, false
	}
	user := User{ID: int64(id), Bot: flags&(1<<14) != 0, Self: flags&(1<<10) != 0}
	if flags&1 != 0 {
		hash, err := r.long()
		if err != nil {
			return user, true
		}
		user.AccessHash = int64(hash)
	}
	if flags&(1<<1) != 0 {
		if v, err := r.string(); err == nil {
			user.FirstName = v
		} else {
			return user, true
		}
	}
	if flags&(1<<2) != 0 {
		if v, err := r.string(); err == nil {
			user.LastName = v
		} else {
			return user, true
		}
	}
	if flags&(1<<3) != 0 {
		if v, err := r.string(); err == nil {
			user.Username = v
		} else {
			return user, true
		}
	}
	if flags&(1<<4) != 0 {
		if v, err := r.string(); err == nil {
			user.Phone = v
		}
	}
	return user, true
}
