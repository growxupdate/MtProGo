package updates

import "sync"

const (
	// DefaultMessageCacheSize keeps the last 1000 messages by default.
	DefaultMessageCacheSize = 1000
	// DefaultPeerCacheSize keeps metadata for the last 1000 peers by default.
	DefaultPeerCacheSize = 1000
)

// Peer describes a minimal cached Telegram peer.
type Peer struct {
	ID         int64
	AccessHash int64
	Kind       string
	Username   string
	FirstName  string
	LastName   string
	Title      string
	Raw        any
}

// CacheSnapshot exposes cache sizes without leaking internal maps/rings.
type CacheSnapshot struct {
	MessageCount int
	MessageLimit int
	PeerCount    int
	PeerLimit    int
}

// MessageCache is a bounded ring cache for recent messages.
type MessageCache struct {
	mu    sync.RWMutex
	limit int
	items []*Message
	pos   int
	full  bool
}

// NewMessageCache creates a bounded message cache. Size <= 0 disables caching.
func NewMessageCache(size int) *MessageCache {
	if size < 0 {
		size = 0
	}
	return &MessageCache{limit: size, items: make([]*Message, 0, size)}
}

// Limit returns maximum number of cached messages.
func (c *MessageCache) Limit() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.limit
}

// Len returns number of messages currently cached.
func (c *MessageCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Add stores a copy of msg in the ring cache.
func (c *MessageCache) Add(msg *Message) {
	if c == nil || msg == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.limit <= 0 {
		return
	}
	clone := *msg
	if len(c.items) < c.limit {
		c.items = append(c.items, &clone)
		return
	}
	c.items[c.pos] = &clone
	c.pos = (c.pos + 1) % c.limit
	c.full = true
}

// Last returns up to n most recent messages in chronological order.
func (c *MessageCache) Last(n int) []*Message {
	if c == nil || n <= 0 {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	ordered := c.orderedLocked()
	if n > len(ordered) {
		n = len(ordered)
	}
	out := make([]*Message, 0, n)
	for _, item := range ordered[len(ordered)-n:] {
		clone := *item
		out = append(out, &clone)
	}
	return out
}

// All returns all cached messages in chronological order.
func (c *MessageCache) All() []*Message {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	ordered := c.orderedLocked()
	out := make([]*Message, 0, len(ordered))
	for _, item := range ordered {
		clone := *item
		out = append(out, &clone)
	}
	return out
}

func (c *MessageCache) orderedLocked() []*Message {
	if len(c.items) == 0 {
		return nil
	}
	if !c.full {
		return append([]*Message(nil), c.items...)
	}
	out := make([]*Message, 0, len(c.items))
	out = append(out, c.items[c.pos:]...)
	out = append(out, c.items[:c.pos]...)
	return out
}

// PeerCache is a bounded cache of known peers.
type PeerCache struct {
	mu    sync.RWMutex
	limit int
	order []int64
	items map[int64]Peer
}

// NewPeerCache creates a bounded peer cache. Size <= 0 disables peer caching.
func NewPeerCache(size int) *PeerCache {
	if size < 0 {
		size = 0
	}
	return &PeerCache{limit: size, items: make(map[int64]Peer)}
}

// Limit returns maximum peer cache size.
func (c *PeerCache) Limit() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.limit
}

// Len returns current peer count.
func (c *PeerCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Put stores or updates a peer.
func (c *PeerCache) Put(peer Peer) {
	if c == nil || peer.ID == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.limit <= 0 {
		return
	}
	if _, exists := c.items[peer.ID]; !exists {
		if len(c.order) >= c.limit {
			old := c.order[0]
			copy(c.order, c.order[1:])
			c.order = c.order[:len(c.order)-1]
			delete(c.items, old)
		}
		c.order = append(c.order, peer.ID)
	}
	c.items[peer.ID] = peer
}

// Get returns a peer by id.
func (c *PeerCache) Get(id int64) (Peer, bool) {
	if c == nil {
		return Peer{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	peer, ok := c.items[id]
	return peer, ok
}

// All returns a copy of cached peers in insertion order.
func (c *PeerCache) All() []Peer {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Peer, 0, len(c.order))
	for _, id := range c.order {
		if peer, ok := c.items[id]; ok {
			out = append(out, peer)
		}
	}
	return out
}
