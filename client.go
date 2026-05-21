package mtprogo

import (
	"context"
	"errors"

	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/mtproto"
	"github.com/growxupdate/MtProGo/updates"
)

// Config configures the base MtProGo client foundation.
type Config struct {
	APIID   int
	APIHash string
	Session SessionStore
	// SessionName is the key used when a SessionStore can hold multiple sessions.
	SessionName string

	// Updates enables handler dispatching. Caching can still work for manual dispatch when disabled.
	Updates bool
	// MessageCacheSize controls how many recent useful messages are kept in RAM.
	// Use 0 to disable message caching. Default: 1000.
	MessageCacheSize int
	// MessageCacheMode controls which messages enter the bounded cache.
	// Default: CacheMatchedMessages, so high-volume bots do not cache irrelevant updates.
	MessageCacheMode updates.MessageCacheMode
	// MessageCacheFilter is used when MessageCacheMode is CacheFilteredMessages.
	MessageCacheFilter updates.MessageFilter
	// PeerCacheSize controls how many peers are kept in RAM.
	// Use 0 to disable peer caching. Default: 1000.
	PeerCacheSize int
	// UpdateQueueSize reserves future async update queue capacity. Default: 256.
	UpdateQueueSize int
}

// MessageCacheMode aliases the updates cache policy type for user-facing options.
type MessageCacheMode = updates.MessageCacheMode

const (
	// CacheDefault uses MtProGo's safe default: cache only matched messages.
	CacheDefault = updates.CacheDefault
	// CacheNone disables message caching.
	CacheNone = updates.CacheNone
	// CacheAllMessages caches every incoming message. Use carefully on high-volume bots.
	CacheAllMessages = updates.CacheAllMessages
	// CacheMatchedMessages caches only messages that match registered handlers. Default.
	CacheMatchedMessages = updates.CacheMatchedMessages
	// CacheFilteredMessages caches only messages accepted by a custom cache filter.
	CacheFilteredMessages = updates.CacheFilteredMessages
)

// Option mutates Config.
type Option func(*Config)

// Client is the base client foundation. The full MTProto runtime builds on
// this type. V12 adds selective cache controls for high-volume bots.
type Client struct {
	config     Config
	dispatcher *updates.Dispatcher
}

// New creates a base MtProGo client foundation.
func New(apiID int, apiHash string, opts ...Option) (*Client, error) {
	if apiID <= 0 {
		return nil, errors.New("api id must be greater than zero")
	}
	if apiHash == "" {
		return nil, errors.New("api hash is required")
	}

	cfg := Config{
		APIID:            apiID,
		APIHash:          apiHash,
		Session:          MemorySession(),
		SessionName:      "default",
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

	return &Client{
		config: cfg,
		dispatcher: updates.NewDispatcher(
			updates.WithUpdates(cfg.Updates),
			updates.WithMessageCacheSize(cfg.MessageCacheSize),
			updates.WithMessageCacheMode(cfg.MessageCacheMode),
			updates.WithMessageCacheFilter(cfg.MessageCacheFilter),
			updates.WithPeerCacheSize(cfg.PeerCacheSize),
			updates.WithUpdateQueueSize(cfg.UpdateQueueSize),
		),
	}, nil
}

// Must panics if err is not nil and otherwise returns value.
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// Must0 panics if err is not nil.
func Must0(err error) {
	if err != nil {
		panic(err)
	}
}

// WithMemorySession uses an in-memory session store.
func WithMemorySession() Option {
	return func(c *Config) {
		c.Session = MemorySession()
	}
}

// WithSessionName sets the logical name used in session stores that support multiple sessions.
func WithSessionName(name string) Option {
	return func(c *Config) {
		if name != "" {
			c.SessionName = name
		}
	}
}

// WithSessionFile uses a plain file-backed session.
func WithSessionFile(path string) Option {
	return func(c *Config) {
		if path != "" {
			c.Session = FileSession(path)
		}
	}
}

// WithEncryptedSessionFile uses an encrypted file-backed session.
func WithEncryptedSessionFile(path, password string) Option {
	return func(c *Config) {
		if path != "" {
			c.Session = EncryptedFileSession(path, password)
		}
	}
}

// WithStringSession loads an MTProto string session into memory. The session can
// be reused without a phone code or bot token authorization when supported by the runtime.
func WithStringSession(value string) Option {
	return func(c *Config) {
		if value == "" {
			return
		}
		sess, err := parseRootStringSession(value)
		if err != nil {
			return
		}
		c.SessionName = "string"
		c.Session = memorySessionWithData("string", sess)
	}
}

// WithSessionStore uses a custom session store.
func WithSessionStore(store SessionStore) Option {
	return func(c *Config) {
		if store != nil {
			c.Session = store
		}
	}
}

// WithUpdates enables or disables handler dispatching for real updates.
// Message and peer caches can still be populated even when this is false.
func WithUpdates(enabled bool) Option {
	return func(c *Config) { c.Updates = enabled }
}

// WithMessageCacheSize sets how many recent messages are kept in RAM.
// Example: WithMessageCacheSize(1000). Use 0 to disable message caching.
func WithMessageCacheSize(size int) Option {
	return func(c *Config) { c.MessageCacheSize = size }
}

// WithMessageCacheMode sets which messages are kept in RAM.
func WithMessageCacheMode(mode MessageCacheMode) Option {
	return func(c *Config) { c.MessageCacheMode = mode }
}

// WithMessageCacheFilter caches only messages accepted by filter.
// It also switches the cache policy to CacheFilteredMessages.
func WithMessageCacheFilter(filter updates.MessageFilter) Option {
	return func(c *Config) {
		c.MessageCacheFilter = filter
		c.MessageCacheMode = updates.CacheFilteredMessages
	}
}

// WithMessageCacheCommands caches only the specified slash commands.
// This is ideal for bots that only care about commands such as /start, /ping, /help.
func WithMessageCacheCommands(commands ...string) Option {
	return WithMessageCacheFilter(filters.Or(commandFilters(commands...)...))
}

func commandFilters(commands ...string) []filters.Filter {
	out := make([]filters.Filter, 0, len(commands))
	for _, command := range commands {
		out = append(out, filters.Command(command))
	}
	return out
}

// WithPeerCacheSize sets how many peers are kept in RAM. Use 0 to disable.
func WithPeerCacheSize(size int) Option {
	return func(c *Config) { c.PeerCacheSize = size }
}

// WithUpdateQueueSize sets future async update queue capacity.
func WithUpdateQueueSize(size int) Option {
	return func(c *Config) { c.UpdateQueueSize = size }
}

// OnMessage registers a message handler.
func (c *Client) OnMessage(filter filters.Filter, handler updates.MessageHandler) {
	c.dispatcher.OnMessage(filter, handler)
}

// Dispatcher returns the update dispatcher.
func (c *Client) Dispatcher() *updates.Dispatcher {
	return c.dispatcher
}

// CacheSnapshot returns memory cache statistics.
func parseRootStringSession(value string) ([]byte, error) {
	// The low-level mtproto package owns the string-session format; this helper
	// keeps option construction error-free while still validating the payload.
	sess, err := mtproto.ParseStringSession(value)
	if err != nil {
		return nil, err
	}
	return sess.MarshalBinary()
}

func (c *Client) CacheSnapshot() updates.CacheSnapshot {
	if c == nil || c.dispatcher == nil {
		return updates.CacheSnapshot{}
	}
	return c.dispatcher.CacheSnapshot()
}

// Run blocks until ctx is cancelled. Full high-level MTProto runtime will replace this
// with network processing in a later release.
func (c *Client) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
