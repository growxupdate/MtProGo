package mtproto

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	constructorAuthImportBotAuthorization = 0x67a3ff2c
	constructorMessagesSendMessage        = 0xfe05dc9a
	constructorInputPeerEmpty             = 0x7f3b18ea
	constructorInputPeerSelf              = 0x7da07ec9
	constructorInputPeerChat              = 0x35a95cb9
	constructorInputPeerUser              = 0xdde8a54c
	constructorInputPeerChannel           = 0x27bcbbfc
)

// EncryptedClient is a small low-level MTProto client bound to one DC and auth key.
// It is intentionally raw: callers pass exact TL payloads or use the helpers below.
type EncryptedClient struct {
	conn  net.Conn
	dc    DCOption
	auth  *AuthKeyResult
	state *encryptedState
}

// DialEncrypted creates a fresh MTProto auth key and opens an encrypted session to dc.
func DialEncrypted(ctx context.Context, dc DCOption) (*EncryptedClient, error) {
	conn, err := dialMTProto(ctx, dc, 35*time.Second)
	if err != nil {
		return nil, err
	}
	auth, err := authKeyOnConn(conn, dc)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &EncryptedClient{
		conn:  conn,
		dc:    dc,
		auth:  auth,
		state: newEncryptedState(auth),
	}, nil
}

// DialEncryptedDefault tries production DCs until an encrypted session is created.
func DialEncryptedDefault(ctx context.Context) (*EncryptedClient, error) {
	var last error
	for _, dc := range DefaultDCOptions {
		client, err := DialEncrypted(ctx, dc)
		if err == nil {
			return client, nil
		}
		last = err
	}
	if last == nil {
		last = errors.New("no default DCs configured")
	}
	return nil, last
}

// Close closes the underlying MTProto TCP connection.
func (c *EncryptedClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// DC returns the Telegram DC endpoint used by this encrypted client.
func (c *EncryptedClient) DC() DCOption { return c.dc }

// AuthKey returns the generated auth key metadata.
func (c *EncryptedClient) AuthKey() *AuthKeyResult { return c.auth }

// Invoke sends a raw TL function body over encrypted MTProto.
func (c *EncryptedClient) Invoke(ctx context.Context, body []byte) (*InvokeResult, error) {
	if c == nil || c.conn == nil || c.state == nil {
		return nil, errors.New("mtproto: nil encrypted client")
	}
	result, err := c.state.invoke(ctx, c.conn, body)
	if err != nil {
		return nil, err
	}
	result.DCID = c.dc.ID
	result.Address = c.dc.Address
	result.AuthKeyID = c.auth.AuthKeyID
	result.ServerSalt = c.state.serverSalt
	result.SessionID = c.state.sessionID
	return result, nil
}

// ImportBotAuthorization logs a bot in over encrypted MTProto using auth.importBotAuthorization.
func (c *EncryptedClient) ImportBotAuthorization(ctx context.Context, apiID int, apiHash, botToken string) (*InvokeResult, error) {
	if apiID <= 0 {
		return nil, errors.New("api id is required")
	}
	if strings.TrimSpace(apiHash) == "" {
		return nil, errors.New("api hash is required")
	}
	if strings.TrimSpace(botToken) == "" {
		return nil, errors.New("bot token is required")
	}
	result, err := c.Invoke(ctx, makeImportBotAuthorizationQuery(apiID, apiHash, botToken))
	if err != nil {
		return nil, err
	}
	result.Message = "auth.importBotAuthorization succeeded over encrypted MTProto"
	return result, nil
}

// ImportBotAuthorizationDefault creates an encrypted session, logs a bot in, and returns the live client.
// If Telegram returns USER_MIGRATE_X, this helper automatically closes the current
// connection, creates a fresh auth key on DC X, and retries the login there.
// The returned client must be closed by the caller.
func ImportBotAuthorizationDefault(ctx context.Context, apiID int, apiHash, botToken string) (*EncryptedClient, *InvokeResult, error) {
	client, err := DialEncryptedDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	result, err := client.ImportBotAuthorization(ctx, apiID, apiHash, botToken)
	if err == nil {
		return client, result, nil
	}

	migrateTo, ok := MigrationDCID(err)
	if !ok {
		_ = client.Close()
		return nil, nil, err
	}
	_ = client.Close()

	migratedClient, migratedResult, migratedErr := importBotAuthorizationOnDC(ctx, migrateTo, apiID, apiHash, botToken)
	if migratedErr != nil {
		return nil, nil, migratedErr
	}
	migratedResult.Message = fmt.Sprintf("auth.importBotAuthorization succeeded after migrating to DC %d", migrateTo)
	return migratedClient, migratedResult, nil
}

func importBotAuthorizationOnDC(ctx context.Context, dcID int, apiID int, apiHash, botToken string) (*EncryptedClient, *InvokeResult, error) {
	options := dcOptionsByID(dcID)
	if len(options) == 0 {
		return nil, nil, fmt.Errorf("mtproto: no default address configured for migrated DC %d", dcID)
	}
	var last error
	for _, dc := range options {
		client, err := DialEncrypted(ctx, dc)
		if err != nil {
			last = err
			continue
		}
		result, err := client.ImportBotAuthorization(ctx, apiID, apiHash, botToken)
		if err == nil {
			return client, result, nil
		}
		_ = client.Close()
		last = err
		if nextDC, ok := MigrationDCID(err); ok && nextDC != dcID {
			return importBotAuthorizationOnDC(ctx, nextDC, apiID, apiHash, botToken)
		}
	}
	if last == nil {
		last = fmt.Errorf("mtproto: failed to connect to migrated DC %d", dcID)
	}
	return nil, nil, last
}

func dcOptionsByID(dcID int) []DCOption {
	out := make([]DCOption, 0, 2)
	for _, dc := range DefaultDCOptions {
		if dc.ID == dcID {
			out = append(out, dc)
		}
	}
	return out
}

// InputPeer is a raw MTProto InputPeer helper used by MessagesSendMessage.
type InputPeer struct {
	kind       string
	id         int64
	accessHash int64
}

// InputPeerSelf returns inputPeerSelf.
func InputPeerSelf() InputPeer { return InputPeer{kind: "self"} }

// InputPeerChat returns inputPeerChat.
func InputPeerChat(chatID int64) InputPeer { return InputPeer{kind: "chat", id: chatID} }

// InputPeerUser returns inputPeerUser. The access hash is required by Telegram.
func InputPeerUser(userID, accessHash int64) InputPeer {
	return InputPeer{kind: "user", id: userID, accessHash: accessHash}
}

// InputPeerChannel returns inputPeerChannel. The access hash is required by Telegram.
func InputPeerChannel(channelID, accessHash int64) InputPeer {
	return InputPeer{kind: "channel", id: channelID, accessHash: accessHash}
}

func (p InputPeer) encode(b *tlBuffer) error {
	switch p.kind {
	case "self":
		b.putInt(constructorInputPeerSelf)
	case "chat":
		if p.id == 0 {
			return errors.New("inputPeerChat requires chat id")
		}
		b.putInt(constructorInputPeerChat)
		b.putLong(uint64(p.id))
	case "user":
		if p.id == 0 || p.accessHash == 0 {
			return errors.New("inputPeerUser requires user id and access hash")
		}
		b.putInt(constructorInputPeerUser)
		b.putLong(uint64(p.id))
		b.putLong(uint64(p.accessHash))
	case "channel":
		if p.id == 0 || p.accessHash == 0 {
			return errors.New("inputPeerChannel requires channel id and access hash")
		}
		b.putInt(constructorInputPeerChannel)
		b.putLong(uint64(p.id))
		b.putLong(uint64(p.accessHash))
	default:
		return errors.New("unknown input peer kind")
	}
	return nil
}

// MessagesSendMessage sends a plain text message with messages.sendMessage.
// The client should already be authorized, for example through ImportBotAuthorization.
func (c *EncryptedClient) MessagesSendMessage(ctx context.Context, peer InputPeer, text string) (*InvokeResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message text is required")
	}
	query, err := makeMessagesSendMessageQuery(peer, text)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, query)
	if err != nil {
		return nil, err
	}
	result.Message = "messages.sendMessage succeeded over encrypted MTProto"
	return result, nil
}

func makeImportBotAuthorizationQuery(apiID int, apiHash, botToken string) []byte {
	var inner tlBuffer
	inner.putInt(constructorAuthImportBotAuthorization)
	inner.putInt(0) // flags:int
	inner.putInt(uint32(int32(apiID)))
	inner.putString(apiHash)
	inner.putString(botToken)
	return wrapWithLayerAndInitConnection(apiID, appVersionV11, inner.bytes())
}

func makeMessagesSendMessageQuery(peer InputPeer, text string) ([]byte, error) {
	var b tlBuffer
	b.putInt(constructorMessagesSendMessage)
	b.putInt(0) // flags: no optional fields
	if err := peer.encode(&b); err != nil {
		return nil, err
	}
	b.putString(text)
	b.putLong(uint64(randomInt64()))
	return b.bytes(), nil
}

func wrapWithLayerAndInitConnection(apiID int, appVersion string, query []byte) []byte {
	var init tlBuffer
	init.putInt(constructorInitConnection)
	init.putInt(0) // flags: no proxy, no params
	init.putInt(uint32(int32(apiID)))
	init.putString("MtProGo")
	init.putString("Go")
	init.putString(appVersion)
	init.putString("en")
	init.putString("")
	init.putString("en")
	init.putRaw(query)

	var invoke tlBuffer
	invoke.putInt(constructorInvokeWithLayer)
	invoke.putInt(uint32(int32(defaultAPILayer)))
	invoke.putRaw(init.bytes())
	return invoke.bytes()
}

func randomInt64() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UnixNano()
	}
	v := int64(binary.LittleEndian.Uint64(b[:]))
	if v == 0 {
		return time.Now().UnixNano()
	}
	return v
}

// ConstructorName returns a small human-readable name for constructors used by the examples.
func ConstructorName(id uint32) string {
	switch id {
	case 0x2ea2c0d4:
		return "auth.authorization"
	case constructorRpcError:
		return "rpc_error"
	case constructorGzipPacked:
		return "gzip_packed"
	case constructorAccountPassword:
		return "account.password"
	case constructorPasswordKDFAlgoSHA256SHA256PBKDF2HMACSHA512Iter100000SHA256ModPow:
		return "passwordKdfAlgoSHA256SHA256PBKDF2HMACSHA512iter100000SHA256ModPow"
	case constructorInputCheckPasswordSRP:
		return "inputCheckPasswordSRP"

	case 0x74ae4240:
		return "updatesTooLong"
	case 0x9015e101:
		return "updateShortMessage"
	case 0x313bc7f8:
		return "updateShortChatMessage"
	case 0x78d4dec1:
		return "updateShort"
	case 0x725b04c3:
		return "updatesCombined"
	case 0x8dca6aa5:
		return "updates"
	case 0xe317af7e:
		return "updateShortSentMessage"
	default:
		return fmt.Sprintf("0x%08x", id)
	}
}

const appVersionV11 = "v11.0.0-dev"
