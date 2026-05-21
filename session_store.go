package mtprogo

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// SessionDeleter is implemented by stores that can delete sessions.
type SessionDeleter interface {
	Delete(ctx context.Context, name string) error
}

type memorySessionStore struct {
	mu   sync.RWMutex
	data map[string]SessionData
}

// MemorySession creates an in-memory session store.
func MemorySession() SessionStore {
	return &memorySessionStore{data: make(map[string]SessionData)}
}

func memorySessionWithData(name string, data []byte) SessionStore {
	store := &memorySessionStore{data: make(map[string]SessionData)}
	store.data[name] = SessionData{Name: name, Data: append([]byte(nil), data...)}
	return store
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

func (m *memorySessionStore) Delete(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, name)
	return nil
}

// FileSession creates a file-backed session store. If path ends with .session,
// it is treated as a fixed file path; otherwise it is treated as a directory and
// the session name becomes <name>.session inside that directory.
func FileSession(path string) SessionStore { return &fileSessionStore{path: path} }

type fileSessionStore struct{ path string }

func (f *fileSessionStore) sessionPath(name string) string {
	path := strings.TrimSpace(f.path)
	if path == "" {
		path = ".mtprogo"
	}
	if strings.HasSuffix(path, ".session") || strings.HasSuffix(path, ".mtprogo") {
		return path
	}
	if name == "" {
		name = "default"
	}
	return filepath.Join(path, name+".session")
}

func (f *fileSessionStore) Load(ctx context.Context, name string) (SessionData, bool, error) {
	select {
	case <-ctx.Done():
		return SessionData{}, false, ctx.Err()
	default:
	}
	path := f.sessionPath(name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SessionData{}, false, nil
	}
	if err != nil {
		return SessionData{}, false, err
	}
	return SessionData{Name: name, Data: data}, true, nil
}

func (f *fileSessionStore) Save(ctx context.Context, data SessionData) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	path := f.sessionPath(data.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data.Data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *fileSessionStore) Delete(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	path := f.sessionPath(name)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// EncryptedFileSession creates a file-backed encrypted session store. It uses
// AES-256-GCM and a PBKDF2-HMAC-SHA256 key derived from password. Encryption is
// done only when saving/loading the session file, so runtime update performance
// is unaffected.
func EncryptedFileSession(path, password string) SessionStore {
	return &encryptedSessionStore{inner: FileSession(path), password: password}
}

type encryptedSessionStore struct {
	inner    SessionStore
	password string
}

const encryptedSessionMagic = "MPGSESS1"
const encryptedSessionIterations = 120000

func (s *encryptedSessionStore) Load(ctx context.Context, name string) (SessionData, bool, error) {
	data, ok, err := s.inner.Load(ctx, name)
	if err != nil || !ok {
		return data, ok, err
	}
	plain, err := decryptSessionBytes(s.password, data.Data)
	if err != nil {
		return SessionData{}, false, err
	}
	data.Data = plain
	return data, true, nil
}

func (s *encryptedSessionStore) Save(ctx context.Context, data SessionData) error {
	sealed, err := encryptSessionBytes(s.password, data.Data)
	if err != nil {
		return err
	}
	return s.inner.Save(ctx, SessionData{Name: data.Name, Data: sealed})
}

func (s *encryptedSessionStore) Delete(ctx context.Context, name string) error {
	if d, ok := s.inner.(SessionDeleter); ok {
		return d.Delete(ctx, name)
	}
	return nil
}

func encryptSessionBytes(password string, plain []byte) ([]byte, error) {
	if password == "" {
		return nil, errors.New("session encryption password is required")
	}
	var salt [16]byte
	var nonce [12]byte
	if _, err := io.ReadFull(rand.Reader, salt[:]); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	key := pbkdf2SHA256([]byte(password), salt[:], encryptedSessionIterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(encryptedSessionMagic)
	_ = binary.Write(&out, binary.LittleEndian, uint32(encryptedSessionIterations))
	out.Write(salt[:])
	out.Write(nonce[:])
	out.Write(gcm.Seal(nil, nonce[:], plain, nil))
	return []byte(base64.RawURLEncoding.EncodeToString(out.Bytes())), nil
}

func decryptSessionBytes(password string, encoded []byte) ([]byte, error) {
	if password == "" {
		return nil, errors.New("session encryption password is required")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return nil, err
	}
	if len(raw) < len(encryptedSessionMagic)+4+16+12+16 || string(raw[:len(encryptedSessionMagic)]) != encryptedSessionMagic {
		return nil, errors.New("invalid encrypted session file")
	}
	off := len(encryptedSessionMagic)
	iterations := int(binary.LittleEndian.Uint32(raw[off:]))
	off += 4
	salt := raw[off : off+16]
	off += 16
	nonce := raw[off : off+12]
	off += 12
	ciphertext := raw[off:]
	key := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	if iter <= 0 {
		iter = 1
	}
	hLen := sha256.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	for block := 1; block <= numBlocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		var idx [4]byte
		binary.BigEndian.PutUint32(idx[:], uint32(block))
		mac.Write(idx[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func clearSession(ctx context.Context, store SessionStore, name string) error {
	if store == nil {
		return nil
	}
	if d, ok := store.(SessionDeleter); ok {
		return d.Delete(ctx, name)
	}
	return fmt.Errorf("session store does not support delete")
}
