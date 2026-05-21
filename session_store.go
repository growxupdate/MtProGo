package mtprogo

import (
	"context"
	"sync"
)

// SessionData stores serialized client session bytes.
type SessionData struct {
	Name string
	Data []byte
}

// SessionStore is the interface used by MtProGo session backends.
type SessionStore interface {
	Load(ctx context.Context, name string) (SessionData, bool, error)
	Save(ctx context.Context, data SessionData) error
}

type memorySessionStore struct {
	mu   sync.RWMutex
	data map[string]SessionData
}

// MemorySession creates an in-memory session store.
func MemorySession() SessionStore {
	return &memorySessionStore{data: make(map[string]SessionData)}
}

func (m *memorySessionStore) Load(ctx context.Context, name string) (SessionData, bool, error) {
	select {
	case <-ctx.Done():
		return SessionData{}, false, ctx.Err()
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.data[name]
	if !ok {
		return SessionData{}, false, nil
	}
	data.Data = append([]byte(nil), data.Data...)
	return data, true, nil
}

func (m *memorySessionStore) Save(ctx context.Context, data SessionData) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	copyData := data
	copyData.Data = append([]byte(nil), data.Data...)
	m.data[data.Name] = copyData
	return nil
}
