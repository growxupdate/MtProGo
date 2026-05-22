package mtprogo

import "time"

// APIID returns the configured API ID.
func (c *Client) APIID() int { return c.config.APIID }

// APIHash returns the configured API hash.
func (c *Client) APIHash() string { return c.config.APIHash }

// SessionName returns the configured session name.
func (c *Client) SessionName() string { return c.config.SessionName }

// UpdatesEnabled reports whether update handlers are enabled.
func (c *Client) UpdatesEnabled() bool { return c.config.Updates }

// MessageCacheSize returns configured message cache limit.
func (c *Client) MessageCacheSize() int { return c.config.MessageCacheSize }

// PeerCacheSize returns configured peer cache limit.
func (c *Client) PeerCacheSize() int { return c.config.PeerCacheSize }

// UpdateQueueSize returns configured update queue capacity.
func (c *Client) UpdateQueueSize() int { return c.config.UpdateQueueSize }

// MessageCacheMode returns the configured message-cache policy.
func (c *Client) MessageCacheMode() MessageCacheMode { return c.config.MessageCacheMode }

// DebugEnabled reports whether lightweight runtime logs are enabled.
func (c *Client) DebugEnabled() bool { return c.config.Debug }

// AutoReconnectEnabled reports whether automatic MTProto reconnect is enabled.
func (c *Client) AutoReconnectEnabled() bool { return c.config.AutoReconnect }

// MaxReconnectAttempts returns configured reconnect attempt limit.
func (c *Client) MaxReconnectAttempts() int { return c.config.MaxReconnectAttempts }

// ReconnectInitialBackoff returns the initial reconnect delay.
func (c *Client) ReconnectInitialBackoff() time.Duration { return c.config.ReconnectInitialBackoff }

// ReconnectMaxBackoff returns the maximum reconnect delay.
func (c *Client) ReconnectMaxBackoff() time.Duration { return c.config.ReconnectMaxBackoff }

// AutoFloodWaitEnabled reports whether FLOOD_WAIT auto sleep is enabled.
func (c *Client) AutoFloodWaitEnabled() bool { return c.config.AutoFloodWait }

// MaxFloodWait returns the maximum FLOOD_WAIT duration that will be auto-slept.
func (c *Client) MaxFloodWait() time.Duration { return c.config.MaxFloodWait }
