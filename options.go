package mtprogo

// APIID returns the configured API ID.
func (c *Client) APIID() int { return c.config.APIID }

// APIHash returns the configured API hash.
func (c *Client) APIHash() string { return c.config.APIHash }

// UpdatesEnabled reports whether update handlers are enabled.
func (c *Client) UpdatesEnabled() bool { return c.config.Updates }

// MessageCacheSize returns configured message cache limit.
func (c *Client) MessageCacheSize() int { return c.config.MessageCacheSize }

// PeerCacheSize returns configured peer cache limit.
func (c *Client) PeerCacheSize() int { return c.config.PeerCacheSize }

// UpdateQueueSize returns configured update queue capacity.
func (c *Client) UpdateQueueSize() int { return c.config.UpdateQueueSize }
