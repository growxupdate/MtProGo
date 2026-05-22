package updates

import "context"

// MessageFilter checks whether a message should be handled.
type MessageFilter func(*Message) bool

// MessageHandler handles a message update.
type MessageHandler func(context.Context, *Message) error

type messageRoute struct {
	filter  MessageFilter
	handler MessageHandler
}

// Dispatcher routes normalized Telegram updates to handlers.
type Dispatcher struct {
	config        DispatcherConfig
	messageRoutes []messageRoute
	messages      *MessageCache
	peers         *PeerCache
}

// NewDispatcher creates a Dispatcher.
func NewDispatcher(opts ...DispatcherOption) *Dispatcher {
	cfg := DefaultDispatcherConfig()
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
	if cfg.MessageCacheMode == CacheDefault {
		cfg.MessageCacheMode = CacheMatchedMessages
	}
	if cfg.MessageCacheMode < CacheDefault || cfg.MessageCacheMode > CacheFilteredMessages {
		cfg.MessageCacheMode = CacheMatchedMessages
	}
	return &Dispatcher{
		config:   cfg,
		messages: NewMessageCache(cfg.MessageCacheSize),
		peers:    NewPeerCache(cfg.PeerCacheSize),
	}
}

// Config returns dispatcher configuration.
func (d *Dispatcher) Config() DispatcherConfig {
	if d == nil {
		return DefaultDispatcherConfig()
	}
	return d.config
}

// UpdatesEnabled reports whether real update dispatching is enabled.
func (d *Dispatcher) UpdatesEnabled() bool {
	return d == nil || d.config.UpdatesEnabled
}

// MessageCacheMode returns the active message-cache policy.
func (d *Dispatcher) MessageCacheMode() MessageCacheMode {
	if d == nil {
		return DefaultDispatcherConfig().MessageCacheMode
	}
	return d.config.MessageCacheMode
}

// MessageCache returns bounded recent-message cache.
func (d *Dispatcher) MessageCache() *MessageCache {
	if d == nil {
		return nil
	}
	return d.messages
}

// PeerCache returns bounded peer cache.
func (d *Dispatcher) PeerCache() *PeerCache {
	if d == nil {
		return nil
	}
	return d.peers
}

// CacheSnapshot returns message/peer cache sizes.
func (d *Dispatcher) CacheSnapshot() CacheSnapshot {
	if d == nil {
		return CacheSnapshot{}
	}
	return CacheSnapshot{
		MessageCount: d.messages.Len(),
		MessageLimit: d.messages.Limit(),
		PeerCount:    d.peers.Len(),
		PeerLimit:    d.peers.Limit(),
		CacheMode:    d.config.MessageCacheMode.String(),
	}
}

// OnMessage registers a message route.
func (d *Dispatcher) OnMessage(filter MessageFilter, handler MessageHandler) {
	if handler == nil {
		return
	}
	if filter == nil {
		filter = func(*Message) bool { return true }
	}
	d.messageRoutes = append(d.messageRoutes, messageRoute{filter: filter, handler: handler})
}

// PutPeer inserts a peer into the peer cache.
func (d *Dispatcher) PutPeer(peer Peer) {
	if d == nil || d.peers == nil {
		return
	}
	d.peers.Put(peer)
}

// DispatchMessage dispatches one message to all matching routes.
// V12 intentionally evaluates filters before caching, so CacheMatchedMessages
// stores only useful messages such as registered commands instead of every noisy
// incoming update. This keeps RAM bounded for very large bots.
func (d *Dispatcher) DispatchMessage(ctx context.Context, msg *Message) error {
	if d == nil || msg == nil {
		return nil
	}
	matched := d.matchingRoutes(msg)
	cacheThis := d.shouldCacheMessage(msg, len(matched) > 0)
	if cacheThis && d.messages != nil {
		d.messages.Add(msg)
	}
	// Peer cache follows useful messages only: cached messages or handled messages.
	// This prevents one million random non-command senders from filling RAM when
	// the bot only cares about /start, /ping, /help, /queue, etc.
	if (cacheThis || len(matched) > 0) && d.peers != nil {
		d.cachePeers(msg)
	}
	if !d.config.UpdatesEnabled {
		return nil
	}
	for _, route := range matched {
		if err := route.handler(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) matchingRoutes(msg *Message) []messageRoute {
	if len(d.messageRoutes) == 0 {
		return nil
	}
	matched := make([]messageRoute, 0, 1)
	for _, route := range d.messageRoutes {
		if route.filter == nil || route.filter(msg) {
			matched = append(matched, route)
		}
	}
	return matched
}

func (d *Dispatcher) shouldCacheMessage(msg *Message, matched bool) bool {
	if d.messages == nil || d.messages.Limit() <= 0 {
		return false
	}
	switch d.config.MessageCacheMode {
	case CacheNone:
		return false
	case CacheAllMessages:
		return true
	case CacheMatchedMessages:
		return matched
	case CacheFilteredMessages:
		return d.config.MessageCacheFilter != nil && d.config.MessageCacheFilter(msg)
	default:
		return matched
	}
}

func (d *Dispatcher) cachePeers(msg *Message) {
	if msg.FromID != 0 {
		d.peers.Put(Peer{ID: msg.FromID, Kind: "user"})
	}
	if msg.ChatID != 0 {
		kind := msg.ChatType
		if kind == "" {
			kind = "chat"
			if msg.ChatID == msg.FromID {
				kind = "user"
			}
		}
		d.peers.Put(Peer{ID: msg.ChatID, Kind: kind})
	}
}
