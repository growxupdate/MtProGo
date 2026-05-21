package mtprogo

import (
	"context"
	"errors"

	"github.com/growxupdate/MtProGo/filters"
	"github.com/growxupdate/MtProGo/updates"
)

// Config configures the base MtProGo client foundation.
type Config struct {
	APIID   int
	APIHash string
	Session SessionStore

	// Updates enables handler dispatching. Caching still works when disabled.
	Updates bool
	// MessageCacheSize controls how many recent messages are kept in RAM.
	// Use 0 to disable message caching. Default: 1000.
	MessageCacheSize int
	// PeerCacheSize controls how many peers are kept in RAM.
	// Use 0 to disable peer caching. Default: 1000.
	PeerCacheSize int
	// UpdateQueueSize reserves future async update queue capacity. Default: 256.
	UpdateQueueSize int
}

// Option mutates Config.
type Option func(*Config)

// Client is the base client foundation. The full MTProto runtime builds on
// this type. V10 adds configurable updates/cache primitives.
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
