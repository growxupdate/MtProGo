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
}

// Option mutates Config.
type Option func(*Config)

// Client is the base client foundation. The full MTProto runtime will build on
// this type. V8 uses it for local dispatching and shared API shape.
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
		APIID:   apiID,
		APIHash: apiHash,
		Session: MemorySession(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return &Client{
		config:     cfg,
		dispatcher: updates.NewDispatcher(),
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

// OnMessage registers a message handler.
func (c *Client) OnMessage(filter filters.Filter, handler updates.MessageHandler) {
	c.dispatcher.OnMessage(filter, handler)
}

// Dispatcher returns the update dispatcher.
func (c *Client) Dispatcher() *updates.Dispatcher {
	return c.dispatcher
}

// Run blocks until ctx is cancelled. Full MTProto runtime will replace this
// with network processing in a later release.
func (c *Client) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
