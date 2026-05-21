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
		APIID:            config.APIID,
		APIHash:          config.APIHash,
		Session:          MemorySession(),
		SessionName:      "mtproto_bot",
		Updates:          true,
		MessageCacheSize: updates.DefaultMessageCacheSize,
		MessageCacheMode: updates.CacheMatchedMessages,
		PeerCacheSize:    updates.DefaultPeerCacheSize,
		UpdateQueueSize:  256,
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
	if client, ok, err := b.loadSession(ctx); err != nil {
		fmt.Println("saved MTProto session load failed; re-authorizing:", err)
	} else if ok {
		b.client = client
		b.loadedSession = true
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

func (b *MTProtoBot) loadSession(ctx context.Context) (*mtproto.EncryptedClient, bool, error) {
	if b == nil || b.options.Session == nil {
		return nil, false, nil
	}
	data, ok, err := b.options.Session.Load(ctx, b.sessionName())
	if err != nil || !ok {
		return nil, ok, err
	}
	sess, err := mtproto.ParseSession(data.Data)
	if err != nil {
		return nil, false, err
	}
	client, err := mtproto.DialEncryptedFromSession(ctx, sess)
	if err != nil {
		return nil, false, err
	}
	return client, true, nil
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
	data, err := sess.MarshalBinary()
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

	state, _, err := b.client.UpdatesGetState(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("MTProto updates state loaded: pts=%d qts=%d date=%d seq=%d\n", state.PTS, state.QTS, state.Date, state.Seq)
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
			fmt.Println("updates.getDifference error:", err)
			if isTemporaryNetworkError(err) {
				fmt.Println("temporary MTProto network error; reconnecting...")
				if recErr := b.reconnect(ctx); recErr != nil {
					fmt.Println("reconnect error:", recErr)
					continue
				}
			}
			continue
		}
		b.rememberPeers(diff.Users)
		for _, msg := range diff.Messages {
			updateMsg := b.mtprotoTextMessageToUpdate(msg)
			if err := b.dispatcher.DispatchMessage(ctx, updateMsg); err != nil {
				fmt.Println("handler error:", err)
			}
		}
		if diff.State != nil {
			state = mergeUpdateState(state, diff.State)
		}
	}
}

// Reply implements updates.ReplySender.
// V12 supports private user replies when updates.getDifference supplied the user access_hash.
func (b *MTProtoBot) Reply(ctx context.Context, chatID int64, _ int, text string) error {
	if b == nil || b.client == nil {
		return errors.New("mtproto bot is not logged in")
	}
	peer, ok := b.getPeer(chatID)
	if !ok || peer.AccessHash == 0 || peer.Kind != "user" {
		return fmt.Errorf("mtproto: no inputPeerUser access_hash cached for chat %d yet", chatID)
	}
	_, err := b.client.MessagesSendMessage(ctx, mtproto.InputPeerUser(peer.ID, peer.AccessHash), text)
	return err
}

// SendMessage sends a text message to a cached private user peer.
func (b *MTProtoBot) SendMessage(ctx context.Context, userID int64, text string) error {
	return b.Reply(ctx, userID, 0, text)
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
		b.peers[peer.ID] = peer
		if b.dispatcher != nil {
			b.dispatcher.PutPeer(updates.Peer{ID: peer.ID, AccessHash: peer.AccessHash, Kind: peer.Kind})
		}
	}
}

func (b *MTProtoBot) getPeer(id int64) (mtproto.PeerRef, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	peer, ok := b.peers[id]
	return peer, ok
}

func (b *MTProtoBot) mtprotoTextMessageToUpdate(msg mtproto.TextMessage) *updates.Message {
	fromID := msg.FromID
	if fromID == 0 {
		// Bot private messages can omit from_id in some update shapes; peer_id is
		// still the user chat, so use it for handler ergonomics and replies.
		fromID = msg.ChatID
	}
	return &updates.Message{
		ID:          int(msg.ID),
		ChatID:      msg.ChatID,
		FromID:      fromID,
		Text:        msg.Text,
		Raw:         msg,
		ReplySender: b,
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
