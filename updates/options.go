package updates

// DispatcherConfig controls update dispatching and memory use.
type DispatcherConfig struct {
	UpdatesEnabled   bool
	MessageCacheSize int
	PeerCacheSize    int
	UpdateQueueSize  int
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
