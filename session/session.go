package session

import (
	"context"
	"os"
	"path/filepath"
)

// FileStore stores session bytes on disk.
type FileStore struct {
	Dir string
}

// NewFileStore creates a file session store.
func NewFileStore(dir string) *FileStore { return &FileStore{Dir: dir} }

// Load loads session data.
func (s *FileStore) Load(ctx context.Context, name string) ([]byte, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	path := filepath.Join(s.Dir, name+".session")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return data, err == nil, err
}

// Save saves session data.
func (s *FileStore) Save(ctx context.Context, name string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, name+".session"), data, 0o600)
}
