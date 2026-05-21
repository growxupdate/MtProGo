package mtprogo

// APIID returns the configured API ID.
func (c *Client) APIID() int { return c.config.APIID }

// APIHash returns the configured API hash.
func (c *Client) APIHash() string { return c.config.APIHash }
