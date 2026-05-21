package updates

// MessageCacheMode controls which incoming messages are stored in RAM.
type MessageCacheMode int

const (
	// CacheDefault uses MtProGo's safe default: CacheMatchedMessages.
	CacheDefault MessageCacheMode = iota
	// CacheNone disables message caching even when a cache size is configured.
	CacheNone
	// CacheAllMessages caches every incoming message until the bounded cache limit.
	CacheAllMessages
	// CacheMatchedMessages caches only messages that match at least one registered handler.
	// This is the default because high-volume bots should not keep irrelevant updates.
	CacheMatchedMessages
	// CacheFilteredMessages caches only messages accepted by MessageCacheFilter.
	CacheFilteredMessages
)

func (m MessageCacheMode) String() string {
	switch m {
	case CacheDefault:
		return "matched"
	case CacheNone:
		return "none"
	case CacheAllMessages:
		return "all"
	case CacheMatchedMessages:
		return "matched"
	case CacheFilteredMessages:
		return "filtered"
	default:
		return "matched"
	}
}

// DispatcherConfig controls update dispatching and memory use.
type DispatcherConfig struct {
	UpdatesEnabled     bool
	MessageCacheSize   int
	PeerCacheSize      int
	UpdateQueueSize    int
	MessageCacheMode   MessageCacheMode
	MessageCacheFilter MessageFilter
}

// DispatcherOption mutates DispatcherConfig.
type DispatcherOption func(*DispatcherConfig)

// DefaultDispatcherConfig returns safe defaults for low-memory environments.
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		UpdatesEnabled:   true,
		MessageCacheSize: DefaultMessageCacheSize,
		PeerCacheSize:    DefaultPeerCacheSize,
		UpdateQueueSize:  256,
		MessageCacheMode: CacheMatchedMessages,
	}
}

// WithUpdates enables or disables update dispatching.
func WithUpdates(enabled bool) DispatcherOption {
	return func(c *DispatcherConfig) { c.UpdatesEnabled = enabled }
}

// WithMessageCacheSize sets the number of recent messages kept in RAM.
// Use 0 to disable message caching.
func WithMessageCacheSize(size int) DispatcherOption {
	return func(c *DispatcherConfig) { c.MessageCacheSize = size }
}

// WithPeerCacheSize sets the number of peers kept in RAM. Use 0 to disable.
func WithPeerCacheSize(size int) DispatcherOption {
	return func(c *DispatcherConfig) { c.PeerCacheSize = size }
}

// WithUpdateQueueSize sets future async update queue capacity.
func WithUpdateQueueSize(size int) DispatcherOption {
	return func(c *DispatcherConfig) { c.UpdateQueueSize = size }
}

// WithMessageCacheMode sets the message cache policy.
func WithMessageCacheMode(mode MessageCacheMode) DispatcherOption {
	return func(c *DispatcherConfig) { c.MessageCacheMode = mode }
}

// WithMessageCacheFilter caches only messages accepted by filter.
// It automatically switches the cache mode to CacheFilteredMessages.
func WithMessageCacheFilter(filter MessageFilter) DispatcherOption {
	return func(c *DispatcherConfig) {
		if filter == nil {
			return
		}
		c.MessageCacheFilter = filter
		c.MessageCacheMode = CacheFilteredMessages
	}
}
