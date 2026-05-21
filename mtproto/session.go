package mtproto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const sessionVersion = 1
const stringSessionPrefix = "MPG1:"

// Session is the portable MTProto session payload used by MtProGo.
// It contains the long-lived auth key plus the DC needed to reconnect without
// repeating the expensive auth-key handshake or bot/user authorization flow.
type Session struct {
	Version    int    `json:"version"`
	Kind       string `json:"kind,omitempty"` // bot, user, or custom
	DCID       int    `json:"dc_id"`
	Address    string `json:"address"`
	AuthKey    []byte `json:"auth_key"`
	AuthKeyID  uint64 `json:"auth_key_id"`
	ServerSalt int64  `json:"server_salt"`
	SessionID  int64  `json:"session_id"`
	TimeOffset int64  `json:"time_offset"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// Valid checks whether the session has enough data to create an encrypted MTProto connection.
func (s *Session) Valid() error {
	if s == nil {
		return errors.New("mtproto: nil session")
	}
	if s.DCID <= 0 {
		return errors.New("mtproto: session dc id is required")
	}
	if s.Address == "" {
		return errors.New("mtproto: session dc address is required")
	}
	if len(s.AuthKey) != 256 {
		return fmt.Errorf("mtproto: session auth key must be 256 bytes, got %d", len(s.AuthKey))
	}
	if s.AuthKeyID == 0 {
		return errors.New("mtproto: session auth key id is required")
	}
	return nil
}

// MarshalBinary serializes the session to stable JSON bytes.
func (s *Session) MarshalBinary() ([]byte, error) {
	if err := s.Valid(); err != nil {
		return nil, err
	}
	copySession := *s
	if copySession.Version == 0 {
		copySession.Version = sessionVersion
	}
	now := time.Now().Unix()
	if copySession.CreatedAt == 0 {
		copySession.CreatedAt = now
	}
	copySession.UpdatedAt = now
	copySession.AuthKey = append([]byte(nil), s.AuthKey...)
	return json.Marshal(copySession)
}

// ParseSession decodes a MtProGo MTProto session from bytes.
func ParseSession(data []byte) (*Session, error) {
	if len(data) == 0 {
		return nil, errors.New("mtproto: empty session data")
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Version == 0 {
		s.Version = sessionVersion
	}
	if err := s.Valid(); err != nil {
		return nil, err
	}
	s.AuthKey = append([]byte(nil), s.AuthKey...)
	return &s, nil
}

// String encodes the session as a compact string session suitable for copying
// between machines. Keep it secret: it contains the auth key.
func (s *Session) String() string {
	encoded, err := s.EncodeString()
	if err != nil {
		return ""
	}
	return encoded
}

// EncodeString encodes the session as a MtProGo string session.
func (s *Session) EncodeString() (string, error) {
	data, err := s.MarshalBinary()
	if err != nil {
		return "", err
	}
	return stringSessionPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}

// ParseStringSession decodes a MtProGo string session.
func ParseStringSession(value string) (*Session, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, stringSessionPrefix) {
		value = strings.TrimPrefix(value, stringSessionPrefix)
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return ParseSession(data)
}

// ExportSession returns the current encrypted MTProto session.
func (c *EncryptedClient) ExportSession(kind string) *Session {
	if c == nil || c.auth == nil || c.state == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().Unix()
	return &Session{
		Version:    sessionVersion,
		Kind:       kind,
		DCID:       c.dc.ID,
		Address:    c.dc.Address,
		AuthKey:    append([]byte(nil), c.auth.AuthKey...),
		AuthKeyID:  c.auth.AuthKeyID,
		ServerSalt: c.state.serverSalt,
		SessionID:  c.state.sessionID,
		TimeOffset: c.state.timeOffset,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// DialEncryptedFromSession opens a fresh TCP connection and resumes encrypted MTProto
// using a previously saved auth key. It does not repeat auth-key generation.
func DialEncryptedFromSession(ctx context.Context, session *Session) (*EncryptedClient, error) {
	if err := session.Valid(); err != nil {
		return nil, err
	}
	dc := DCOption{ID: session.DCID, Address: session.Address}
	conn, err := dialMTProto(ctx, dc, 20*time.Second)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	auth := &AuthKeyResult{
		DCID:       session.DCID,
		Address:    session.Address,
		AuthKey:    append([]byte(nil), session.AuthKey...),
		AuthKeyID:  session.AuthKeyID,
		ServerSalt: session.ServerSalt,
		TimeOffset: session.TimeOffset,
		Message:    "auth key loaded from saved session",
	}
	return &EncryptedClient{
		conn:  conn,
		dc:    dc,
		auth:  auth,
		state: newEncryptedStateFromSession(session),
	}, nil
}

func newEncryptedStateFromSession(session *Session) *encryptedState {
	state := &encryptedState{
		authKey:    append([]byte(nil), session.AuthKey...),
		authKeyID:  session.AuthKeyID,
		serverSalt: session.ServerSalt,
		timeOffset: session.TimeOffset,
	}
	// Use a new MTProto session_id for every process/connection. We still store
	// the last session_id for observability, but a fresh ID avoids duplicate
	// session semantics when the old process was not shut down cleanly.
	state.sessionID = randomInt64()
	return state
}

// Connection returns the underlying net.Conn for internal tests.
func (c *EncryptedClient) connection() net.Conn {
	if c == nil {
		return nil
	}
	return c.conn
}
