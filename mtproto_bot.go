package mtprogo

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/mtproto"
	"github.com/growxupdate/MtProGo/updates"
)

// MTProtoBotConfig configures the pure MTProto bot runtime.
type MTProtoBotConfig struct {
	APIID   int
	APIHash string
	Token   string

	// PollInterval controls updates.getDifference polling. Default: 2 seconds.
	PollInterval time.Duration
	// PTSLimit is passed to updates.getDifference. Default: 100.
	PTSLimit int32
}

// MTProtoBot is an experimental pure MTProto bot runtime.
// It logs in with auth.importBotAuthorization and receives text updates through
// updates.getDifference. The Bot API runtime remains available as Bot.
type MTProtoBot struct {
	config        MTProtoBotConfig
	options       Config
	client        *mtproto.EncryptedClient
	dispatcher    *updates.Dispatcher
	loadedSession bool
	lastState     *mtproto.UpdatesState

	mu    sync.RWMutex
	peers map[int64]mtproto.PeerRef
}

// NewMTProtoBot creates a pure MTProto bot runtime.
func NewMTProtoBot(config MTProtoBotConfig, opts ...Option) (*MTProtoBot, error) {
	if config.APIID <= 0 {
		return nil, errors.New("api id is required")
	}
	if config.APIHash == "" {
		return nil, errors.New("api hash is required")
	}
	if config.Token == "" {
		return nil, errors.New("bot token is required")
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.PTSLimit <= 0 {
		config.PTSLimit = 100
	}

	cfg := Config{
		APIID:                   config.APIID,
		APIHash:                 config.APIHash,
		Session:                 MemorySession(),
		SessionName:             "mtproto_bot",
		Updates:                 true,
		MessageCacheSize:        updates.DefaultMessageCacheSize,
		MessageCacheMode:        updates.CacheMatchedMessages,
		PeerCacheSize:           updates.DefaultPeerCacheSize,
		UpdateQueueSize:         256,
		AutoReconnect:           true,
		ReconnectInitialBackoff: time.Second,
		ReconnectMaxBackoff:     30 * time.Second,
		MaxFloodWait:            60 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.MessageCacheSize < 0 {
		cfg.MessageCacheSize = 0
	}
	if cfg.PeerCacheSize < 0 {
		cfg.PeerCacheSize = 0
	}
	if cfg.UpdateQueueSize < 0 {
		cfg.UpdateQueueSize = 0
	}
	if cfg.ReconnectInitialBackoff <= 0 {
		cfg.ReconnectInitialBackoff = time.Second
	}
	if cfg.ReconnectMaxBackoff <= 0 {
		cfg.ReconnectMaxBackoff = 30 * time.Second
	}
	if cfg.MaxFloodWait <= 0 {
		cfg.MaxFloodWait = 60 * time.Second
	}

	return &MTProtoBot{
		config:  config,
		options: cfg,
		dispatcher: updates.NewDispatcher(
			updates.WithUpdates(cfg.Updates),
			updates.WithMessageCacheSize(cfg.MessageCacheSize),
			updates.WithMessageCacheMode(cfg.MessageCacheMode),
			updates.WithMessageCacheFilter(cfg.MessageCacheFilter),
			updates.WithPeerCacheSize(cfg.PeerCacheSize),
			updates.WithUpdateQueueSize(cfg.UpdateQueueSize),
		),
		peers: make(map[int64]mtproto.PeerRef),
	}, nil
}

// OnMessage registers a message handler.
func (b *MTProtoBot) OnMessage(filter filters.Filter, handler updates.MessageHandler) {
	b.dispatcher.OnMessage(filter, handler)
}

// Dispatcher returns the runtime dispatcher.
func (b *MTProtoBot) Dispatcher() *updates.Dispatcher { return b.dispatcher }

// CacheSnapshot returns message and peer cache stats.
func (b *MTProtoBot) CacheSnapshot() updates.CacheSnapshot { return b.dispatcher.CacheSnapshot() }

// Login opens a saved MTProto session when available; otherwise it creates a new
// auth key, authorizes the bot, and saves the session for future runs.
func (b *MTProtoBot) Login(ctx context.Context) error {
	if client, state, peers, ok, err := b.loadSession(ctx); err != nil {
		fmt.Println("saved MTProto session load failed; re-authorizing:", err)
	} else if ok {
		b.client = client
		b.loadedSession = true
		b.lastState = state
		b.rememberPeers(peers)
		fmt.Printf("MtProGo MTProto bot session loaded on DC %d (%s)\n", client.DC().ID, client.DC().Address)
		return nil
	}

	client, result, err := mtproto.ImportBotAuthorizationDefault(ctx, b.config.APIID, b.config.APIHash, b.config.Token)
	if err != nil {
		return err
	}
	b.client = client
	b.loadedSession = false
	fmt.Printf("MtProGo MTProto bot authorized on DC %d (%s), result=%s\n", result.DCID, result.Address, mtproto.ConstructorName(result.Constructor))
	if err := b.SaveSession(ctx); err != nil {
		fmt.Println("session save warning:", err)
	}
	return nil
}

func (b *MTProtoBot) sessionName() string {
	if b.options.SessionName != "" {
		return b.options.SessionName
	}
	return "mtproto_bot"
}

func (b *MTProtoBot) loadSession(ctx context.Context) (*mtproto.EncryptedClient, *mtproto.UpdatesState, []mtproto.PeerRef, bool, error) {
	if b == nil || b.options.Session == nil {
		return nil, nil, nil, false, nil
	}
	data, ok, err := b.options.Session.Load(ctx, b.sessionName())
	if err != nil || !ok {
		return nil, nil, nil, ok, err
	}
	bundle, err := parseRuntimeSession(data.Data)
	if err != nil {
		return nil, nil, nil, false, err
	}
	client, err := mtproto.DialEncryptedFromSession(ctx, bundle.MTProto)
	if err != nil {
		return nil, nil, nil, false, err
	}
	return client, bundle.State, bundle.Peers, true, nil
}

// SaveSession writes the currently connected MTProto auth key/session metadata.
func (b *MTProtoBot) SaveSession(ctx context.Context) error {
	if b == nil || b.client == nil || b.options.Session == nil {
		return nil
	}
	sess := b.client.ExportSession("bot")
	if sess == nil {
		return nil
	}
	data, err := marshalRuntimeSession("bot", sess, b.snapshotState(), b.snapshotPeers())
	if err != nil {
		return err
	}
	return b.options.Session.Save(ctx, SessionData{Name: b.sessionName(), Data: data})
}

// ExportStringSession returns a copy-pasteable string session for the current MTProto bot session.
func (b *MTProtoBot) ExportStringSession() (string, error) {
	if b == nil || b.client == nil {
		return "", errors.New("mtproto bot is not logged in")
	}
	sess := b.client.ExportSession("bot")
	if sess == nil {
		return "", errors.New("mtproto bot session is unavailable")
	}
	return sess.EncodeString()
}

// ClearSession removes the saved session from the configured session store.
func (b *MTProtoBot) ClearSession(ctx context.Context) error {
	if b == nil || b.options.Session == nil {
		return nil
	}
	return clearSession(ctx, b.options.Session, b.sessionName())
}

// Close closes the MTProto connection.
func (b *MTProtoBot) Close() error {
	if b == nil || b.client == nil {
		return nil
	}
	return b.client.Close()
}

func (b *MTProtoBot) reconnect(ctx context.Context) error {
	if err := b.SaveSession(ctx); err != nil && b.options.Debug {
		fmt.Println("session save warning before reconnect:", err)
	}
	_ = b.Close()
	b.client = nil
	b.loadedSession = false
	return b.Login(ctx)
}

func isTemporaryNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

// Run logs in and starts polling updates.getDifference.
func (b *MTProtoBot) Run(ctx context.Context) error {
	if b.client == nil {
		if err := b.Login(ctx); err != nil {
			return err
		}
	}
	defer b.Close()
	defer func() {
		if err := b.SaveSession(context.Background()); err != nil && b.options.Debug {
			fmt.Println("session save warning on shutdown:", err)
		}
	}()

	state := b.snapshotState()
	if state != nil {
		fmt.Printf("MTProto updates state loaded from session: pts=%d qts=%d date=%d seq=%d\n", state.PTS, state.QTS, state.Date, state.Seq)
	} else {
		freshState, _, err := b.client.UpdatesGetState(ctx)
		if err != nil {
			return err
		}
		state = freshState
		b.setState(state)
		fmt.Printf("MTProto updates state loaded: pts=%d qts=%d date=%d seq=%d\n", state.PTS, state.QTS, state.Date, state.Seq)
	}
	if err := b.SaveSession(ctx); err != nil {
		fmt.Println("session save warning:", err)
	}

	if !b.options.Updates {
		fmt.Println("updates disabled; MTProto bot is logged in and waiting for context cancellation")
		<-ctx.Done()
		return ctx.Err()
	}

	ticker := time.NewTicker(b.config.PollInterval)
	defer ticker.Stop()
	fmt.Println("Listening for MTProto updates...")
	consecutiveErrors := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		diff, _, err := b.client.UpdatesGetDifference(ctx, *state, b.config.PTSLimit)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if b.handleFloodWait(ctx, err) {
				continue
			}
			fmt.Println("updates.getDifference error:", err)
			if b.options.AutoReconnect && isTemporaryNetworkError(err) {
				consecutiveErrors++
				if err := b.reconnectWithBackoff(ctx, consecutiveErrors); err != nil {
					fmt.Println("reconnect error:", err)
				}
			}
			continue
		}
		consecutiveErrors = 0
		b.rememberPeers(diff.Peers)
		for _, msg := range diff.Messages {
			b.rememberPeers(b.peersFromTextMessage(msg))
			updateMsg := b.mtprotoTextMessageToUpdate(msg)
			if err := b.dispatcher.DispatchMessage(ctx, updateMsg); err != nil {
				fmt.Println("handler error:", err)
			}
		}
		if diff.State != nil {
			state = mergeUpdateState(state, diff.State)
			b.setState(state)
		}
		if err := b.SaveSession(ctx); err != nil && b.options.Debug {
			fmt.Println("session save warning:", err)
		}
	}
}

func (b *MTProtoBot) handleFloodWait(ctx context.Context, err error) bool {
	wait, ok := mtproto.FloodWaitDuration(err)
	if !ok || !b.options.AutoFloodWait || wait > b.options.MaxFloodWait {
		return false
	}
	fmt.Printf("FLOOD_WAIT: sleeping %s before retry\n", wait)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
		return true
	}
}

func (b *MTProtoBot) reconnectWithBackoff(ctx context.Context, attempt int) error {
	if b.options.MaxReconnectAttempts > 0 && attempt > b.options.MaxReconnectAttempts {
		return fmt.Errorf("mtproto: reconnect attempt limit reached: %d", b.options.MaxReconnectAttempts)
	}
	delay := reconnectBackoff(attempt, b.options.ReconnectInitialBackoff, b.options.ReconnectMaxBackoff)
	if delay > 0 {
		if b.options.Debug {
			fmt.Printf("reconnecting after %s (attempt %d)\n", delay, attempt)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return b.reconnect(ctx)
}

func reconnectBackoff(attempt int, initial, max time.Duration) time.Duration {
	if attempt <= 0 {
		return 0
	}
	if initial <= 0 {
		initial = time.Second
	}
	if max <= 0 {
		max = 30 * time.Second
	}
	delay := initial
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}
	if delay > max {
		return max
	}
	return delay
}

func (b *MTProtoBot) setState(state *mtproto.UpdatesState) {
	if state == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	copyState := *state
	copyState.Raw = append([]byte(nil), state.Raw...)
	b.lastState = &copyState
}

func (b *MTProtoBot) snapshotState() *mtproto.UpdatesState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.lastState == nil {
		return nil
	}
	copyState := *b.lastState
	copyState.Raw = append([]byte(nil), b.lastState.Raw...)
	return &copyState
}

func (b *MTProtoBot) snapshotPeers() []mtproto.PeerRef {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.peers) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(b.peers))
	out := make([]mtproto.PeerRef, 0, len(b.peers))
	for _, peer := range b.peers {
		key := fmt.Sprintf("%s:%d", peer.Kind, peer.ID)
		if _, ok := seen[key]; ok || peer.ID == 0 {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, peer)
	}
	return out
}

// Reply implements updates.ReplySender.
// It can send to cached private users, basic groups, and channels/supergroups.
func (b *MTProtoBot) Reply(ctx context.Context, chatID int64, _ int, text string) error {
	return b.SendMessage(ctx, chatID, text)
}

// SendMessage sends a text message to a cached private/group/channel peer.
func (b *MTProtoBot) SendMessage(ctx context.Context, chatID int64, text string) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok {
		return fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesSendMessageToPeer(callCtx, peer, text)
		return err
	})
}

// EditMessage edits a cached private/group/channel text message when Telegram permits it.
func (b *MTProtoBot) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok {
		return fmt.Errorf("mtproto: peer %d is not cached yet", chatID)
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesEditMessageToPeer(callCtx, peer, int32(messageID), text)
		return err
	})
}

// DeleteMessages deletes messages in cached private/group contexts when Telegram permits it.
func (b *MTProtoBot) DeleteMessages(ctx context.Context, _ int64, messageIDs ...int) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	ids := make([]int32, 0, len(messageIDs))
	for _, id := range messageIDs {
		if id > 0 {
			ids = append(ids, int32(id))
		}
	}
	return b.withFloodWait(ctx, func(callCtx context.Context) error {
		_, err := b.client.MessagesDeleteMessages(callCtx, true, ids...)
		return err
	})
}

func (b *MTProtoBot) withFloodWait(ctx context.Context, fn func(context.Context) error) error {
	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		wait, ok := mtproto.FloodWaitDuration(err)
		if !ok || !b.options.AutoFloodWait || wait > b.options.MaxFloodWait {
			return err
		}
		fmt.Printf("FLOOD_WAIT: sleeping %s before retry\n", wait)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (b *MTProtoBot) rememberPeers(peers []mtproto.PeerRef) {
	if len(peers) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, peer := range peers {
		if peer.ID == 0 {
			continue
		}
		peer = b.mergePeerLocked(peer)
		b.storePeerLocked(peer)
	}
}

func (b *MTProtoBot) mergePeerLocked(peer mtproto.PeerRef) mtproto.PeerRef {
	if peer.ID == 0 {
		return peer
	}
	keys := []int64{peer.ID, publicPeerID(peer)}
	for _, key := range keys {
		if key == 0 {
			continue
		}
		old, ok := b.peers[key]
		if !ok {
			continue
		}
		if peer.AccessHash == 0 {
			peer.AccessHash = old.AccessHash
		}
		if peer.Kind == "" || (peer.Kind == "channel" && old.Kind == "supergroup") {
			// Message.peer_id only says PeerChannel; the chats vector tells us
			// whether it is really a supergroup. Keep the more specific cached kind.
			peer.Kind = old.Kind
		}
		if peer.Username == "" {
			peer.Username = old.Username
		}
		if peer.Title == "" {
			peer.Title = old.Title
		}
	}
	return peer
}

func (b *MTProtoBot) storePeerLocked(peer mtproto.PeerRef) {
	if peer.ID == 0 {
		return
	}
	b.peers[peer.ID] = peer
	publicID := publicPeerID(peer)
	if publicID != 0 {
		b.peers[publicID] = peer
	}
	if b.dispatcher != nil {
		b.dispatcher.PutPeer(updates.Peer{ID: publicID, AccessHash: peer.AccessHash, Kind: peer.Kind, Username: peer.Username, Title: peer.Title})
	}
}

func publicPeerID(peer mtproto.PeerRef) int64 {
	return updates.NormalizeChatID(peer.ID, updateChatType(peer.Kind))
}

func (b *MTProtoBot) getPeer(id int64) (mtproto.PeerRef, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if peer, ok := b.peers[id]; ok {
		return peer, true
	}
	rawID := updates.RawChatID(id)
	peer, ok := b.peers[rawID]
	return peer, ok
}

func (b *MTProtoBot) mtprotoTextMessageToUpdate(msg mtproto.TextMessage) *updates.Message {
	fromID := msg.FromID
	if fromID == 0 {
		// Bot private messages can omit from_id in some update shapes; peer_id is
		// still the user chat, so use it for handler ergonomics and replies.
		fromID = msg.ChatID
	}
	chatKind := msg.ChatKind
	if peer, ok := b.getPeer(msg.ChatID); ok && peer.Kind != "" {
		// The Message peer only distinguishes PeerChannel. The chats vector can
		// identify whether that PeerChannel is a supergroup or broadcast channel.
		chatKind = peer.Kind
	}
	chatType := updateChatType(chatKind)
	if chatType == "" && msg.ChatID == fromID {
		chatType = updates.ChatPrivate
	}
	publicChatID := updates.NormalizeChatID(msg.ChatID, chatType)
	return &updates.Message{
		ID:          int(msg.ID),
		ChatID:      publicChatID,
		RawChatID:   msg.ChatID,
		FromID:      fromID,
		ChatType:    chatType,
		Text:        msg.Text,
		Raw:         msg,
		ReplySender: b,
		Editor:      b,
		Deleter:     b,
	}
}

func (b *MTProtoBot) peersFromTextMessage(msg mtproto.TextMessage) []mtproto.PeerRef {
	peers := make([]mtproto.PeerRef, 0, 2)
	if msg.ChatID != 0 && msg.ChatKind != "" {
		peers = append(peers, mtproto.PeerRef{ID: msg.ChatID, Kind: msg.ChatKind})
	}
	if msg.FromID != 0 && msg.FromKind != "" {
		peers = append(peers, mtproto.PeerRef{ID: msg.FromID, Kind: msg.FromKind})
	}
	return peers
}

func updateChatType(kind string) string {
	switch kind {
	case "user":
		return updates.ChatPrivate
	case "chat":
		return updates.ChatGroup
	case "supergroup":
		return updates.ChatSupergroup
	case "channel":
		return updates.ChatChannel
	default:
		return ""
	}
}

func mergeUpdateState(old, next *mtproto.UpdatesState) *mtproto.UpdatesState {
	if old == nil {
		return next
	}
	if next == nil {
		return old
	}
	merged := *old
	if next.PTS != 0 {
		merged.PTS = next.PTS
	}
	if next.QTS != 0 {
		merged.QTS = next.QTS
	}
	if next.Date != 0 {
		merged.Date = next.Date
	}
	if next.Seq != 0 {
		merged.Seq = next.Seq
	}
	if next.UnreadCount != 0 {
		merged.UnreadCount = next.UnreadCount
	}
	if len(next.Raw) != 0 {
		merged.Raw = append([]byte(nil), next.Raw...)
	}
	return &merged
}
