package mtproto

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	constructorRpcResult          = 0xf35c6d01
	constructorRpcError           = 0x2144ca19
	constructorMsgContainer       = 0x73f1f8dc
	constructorBadMsgNotification = 0xa7eff811
	constructorBadServerSalt      = 0xedab447b
	constructorNewSessionCreated  = 0x9ec20908
	constructorMsgsAck            = 0x62d6b459
	constructorGzipPacked         = 0x3072cfa1
	constructorHelpGetConfig      = 0xc4f9186b
	constructorInvokeWithLayer    = 0xda9b0d0d
	constructorInitConnection     = 0xc1cd5ea9
)

const defaultAPILayer = 214

// InvokeResult is the result of a successful encrypted MTProto invocation.
type InvokeResult struct {
	DCID              int
	Address           string
	AuthKeyID         uint64
	ServerSalt        int64
	SessionID         int64
	RequestMessageID  int64
	ResponseMessageID int64
	Constructor       uint32
	Body              []byte
	GzipPacked        bool
	Message           string
}

// AuthKeyIDHex returns the encrypted MTProto auth key ID in hexadecimal.
func (r InvokeResult) AuthKeyIDHex() string {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], r.AuthKeyID)
	return hex.EncodeToString(b[:])
}

// ConstructorHex returns the response constructor in the usual 0x-prefixed format.
func (r InvokeResult) ConstructorHex() string { return fmt.Sprintf("0x%08x", r.Constructor) }

// RPCError represents an rpc_error returned by Telegram.
type RPCError struct {
	Code    int32
	Message string
}

func (e *RPCError) Error() string {
	if e == nil {
		return "mtproto rpc_error"
	}
	return fmt.Sprintf("mtproto rpc_error %d: %s", e.Code, e.Message)
}

// MigrationDCID extracts the target DC from errors such as USER_MIGRATE_5,
// PHONE_MIGRATE_4, NETWORK_MIGRATE_2, or FILE_MIGRATE_3.
func MigrationDCID(err error) (int, bool) {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return 0, false
	}
	msg := cleanRPCErrorMessage(rpcErr.Message)
	prefixes := []string{"USER_MIGRATE_", "PHONE_MIGRATE_", "NETWORK_MIGRATE_", "FILE_MIGRATE_"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(msg, prefix) {
			n, convErr := strconv.Atoi(strings.TrimPrefix(msg, prefix))
			if convErr == nil && n > 0 {
				return n, true
			}
		}
	}
	return 0, false
}

// HelpGetConfigDefault creates an auth key and calls help.getConfig over encrypted MTProto.
func HelpGetConfigDefault(ctx context.Context, apiID int) (*InvokeResult, error) {
	var last error
	for _, dc := range DefaultDCOptions {
		result, err := HelpGetConfig(ctx, dc, apiID)
		if err == nil {
			return result, nil
		}
		last = err
	}
	if last == nil {
		last = errors.New("no default DCs configured")
	}
	return nil, last
}

// HelpGetConfig creates an auth key and calls help.getConfig over encrypted MTProto.
// When apiID is greater than zero, the request is wrapped with invokeWithLayer/initConnection.
func HelpGetConfig(ctx context.Context, dc DCOption, apiID int) (*InvokeResult, error) {
	conn, err := dialMTProto(ctx, dc, 35*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	auth, err := authKeyOnConn(conn, dc)
	if err != nil {
		return nil, err
	}

	state := newEncryptedState(auth)
	query := makeHelpGetConfigQuery(apiID)
	result, err := state.invoke(ctx, conn, query)
	if err != nil {
		return nil, err
	}
	result.DCID = dc.ID
	result.Address = dc.Address
	result.AuthKeyID = auth.AuthKeyID
	result.ServerSalt = state.serverSalt
	result.SessionID = state.sessionID
	result.Message = "help.getConfig received over encrypted MTProto"
	return result, nil
}

type encryptedState struct {
	authKey    []byte
	authKeyID  uint64
	serverSalt int64
	sessionID  int64
	timeOffset int64
	seq        int32
	lastMsgID  int64
}

func newEncryptedState(auth *AuthKeyResult) *encryptedState {
	var sid [8]byte
	_, _ = rand.Read(sid[:])
	return &encryptedState{
		authKey:    append([]byte(nil), auth.AuthKey...),
		authKeyID:  auth.AuthKeyID,
		serverSalt: auth.ServerSalt,
		sessionID:  int64(binary.LittleEndian.Uint64(sid[:])),
		timeOffset: auth.TimeOffset,
	}
}

func (s *encryptedState) nextSeqNo(contentRelated bool) int32 {
	if !contentRelated {
		return s.seq * 2
	}
	seq := s.seq*2 + 1
	s.seq++
	return seq
}

func (s *encryptedState) nextMsgID() int64 {
	for {
		now := time.Now().Add(time.Duration(s.timeOffset) * time.Second).UnixNano()
		seconds := now / int64(time.Second)
		nanos := now % int64(time.Second)
		msgID := (seconds << 32) | ((nanos << 32) / int64(time.Second))
		msgID &= ^int64(3)
		if msgID > s.lastMsgID {
			s.lastMsgID = msgID
			return msgID
		}
		s.lastMsgID += 4
		return s.lastMsgID
	}
}

func (s *encryptedState) invoke(ctx context.Context, conn net.Conn, body []byte) (*InvokeResult, error) {
	requestMsgID := s.nextMsgID()
	seqNo := s.nextSeqNo(true)
	packet, err := s.buildEncryptedPacket(requestMsgID, seqNo, body)
	if err != nil {
		return nil, err
	}
	if err := abridgedWrite(conn, packet); err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetReadDeadline(deadline)
	}
	for {
		payload, err := abridgedRead(conn)
		if err != nil {
			return nil, err
		}
		msg, err := s.parseEncryptedPacket(payload)
		if err != nil {
			return nil, err
		}
		result, done, err := s.handleMessage(msg, requestMsgID)
		if err != nil {
			return nil, err
		}
		if done {
			result.RequestMessageID = requestMsgID
			return result, nil
		}
	}
}

func (s *encryptedState) buildEncryptedPacket(msgID int64, seqNo int32, body []byte) ([]byte, error) {
	var plain []byte
	plain = binary.LittleEndian.AppendUint64(plain, uint64(s.serverSalt))
	plain = binary.LittleEndian.AppendUint64(plain, uint64(s.sessionID))
	plain = binary.LittleEndian.AppendUint64(plain, uint64(msgID))
	plain = binary.LittleEndian.AppendUint32(plain, uint32(seqNo))
	plain = binary.LittleEndian.AppendUint32(plain, uint32(len(body)))
	plain = append(plain, body...)

	paddingLen := 12 + ((16 - ((len(plain) + 12) % 16)) % 16)
	padding := make([]byte, paddingLen)
	if _, err := rand.Read(padding); err != nil {
		return nil, err
	}
	plain = append(plain, padding...)

	msgKeyFull := sha256Bytes(s.authKey[88:120], plain)
	msgKey := msgKeyFull[8:24]
	key, iv := mtproto2AESKeyIV(s.authKey, msgKey, 0)
	ciphertext, err := aesIGEEncrypt(plain, key, iv)
	if err != nil {
		return nil, err
	}

	var out []byte
	out = binary.LittleEndian.AppendUint64(out, s.authKeyID)
	out = append(out, msgKey...)
	out = append(out, ciphertext...)
	return out, nil
}

type encryptedMessage struct {
	Salt      int64
	SessionID int64
	MsgID     int64
	SeqNo     int32
	Body      []byte
}

func (s *encryptedState) parseEncryptedPacket(payload []byte) (*encryptedMessage, error) {
	if len(payload) < 24 || (len(payload)-24)%16 != 0 {
		return nil, errors.New("mtproto: invalid encrypted packet length")
	}
	r := newTLReader(payload)
	authKeyID, err := r.long()
	if err != nil {
		return nil, err
	}
	if authKeyID != s.authKeyID {
		return nil, fmt.Errorf("mtproto: auth key id mismatch: got %x", authKeyID)
	}
	msgKey, err := r.raw(16)
	if err != nil {
		return nil, err
	}
	ciphertext := append([]byte(nil), r.remaining()...)
	key, iv := mtproto2AESKeyIV(s.authKey, msgKey, 8)
	plain, err := aesIGEDecrypt(ciphertext, key, iv)
	if err != nil {
		return nil, err
	}
	wantFull := sha256Bytes(s.authKey[96:128], plain)
	if !equalBytes(msgKey, wantFull[8:24]) {
		return nil, errors.New("mtproto: inbound msg_key verification failed")
	}
	pr := newTLReader(plain)
	salt, err := pr.long()
	if err != nil {
		return nil, err
	}
	sessionID, err := pr.long()
	if err != nil {
		return nil, err
	}
	msgID, err := pr.long()
	if err != nil {
		return nil, err
	}
	seqNo, err := pr.int()
	if err != nil {
		return nil, err
	}
	length, err := pr.int()
	if err != nil {
		return nil, err
	}
	body, err := pr.raw(int(length))
	if err != nil {
		return nil, err
	}
	return &encryptedMessage{
		Salt:      int64(salt),
		SessionID: int64(sessionID),
		MsgID:     int64(msgID),
		SeqNo:     int32(seqNo),
		Body:      append([]byte(nil), body...),
	}, nil
}

func (s *encryptedState) handleMessage(msg *encryptedMessage, requestMsgID int64) (*InvokeResult, bool, error) {
	if len(msg.Body) < 4 {
		return nil, false, errors.New("mtproto: empty encrypted message body")
	}
	r := newTLReader(msg.Body)
	constructor, err := r.int()
	if err != nil {
		return nil, false, err
	}
	switch constructor {
	case constructorMsgContainer:
		count, err := r.int()
		if err != nil {
			return nil, false, err
		}
		for i := 0; i < int(count); i++ {
			childMsgID, err := r.long()
			if err != nil {
				return nil, false, err
			}
			childSeqNo, err := r.int()
			if err != nil {
				return nil, false, err
			}
			childLen, err := r.int()
			if err != nil {
				return nil, false, err
			}
			childBody, err := r.raw(int(childLen))
			if err != nil {
				return nil, false, err
			}
			for r.off%4 != 0 && r.off < len(r.buf) {
				r.off++
			}
			child := &encryptedMessage{Salt: msg.Salt, SessionID: msg.SessionID, MsgID: int64(childMsgID), SeqNo: int32(childSeqNo), Body: append([]byte(nil), childBody...)}
			result, done, err := s.handleMessage(child, requestMsgID)
			if err != nil || done {
				return result, done, err
			}
		}
		return nil, false, nil
	case constructorRpcResult:
		reqID, err := r.long()
		if err != nil {
			return nil, false, err
		}
		resultBody := append([]byte(nil), r.remaining()...)
		if int64(reqID) != requestMsgID {
			return nil, false, nil
		}
		return parseRPCResultObject(msg.MsgID, resultBody)
	case constructorNewSessionCreated:
		if _, err := r.long(); err != nil { // first_msg_id
			return nil, false, err
		}
		if _, err := r.long(); err != nil { // unique_id
			return nil, false, err
		}
		newSalt, err := r.long()
		if err != nil {
			return nil, false, err
		}
		s.serverSalt = int64(newSalt)
		return nil, false, nil
	case constructorBadServerSalt:
		if _, err := r.long(); err != nil { // bad_msg_id
			return nil, false, err
		}
		if _, err := r.int(); err != nil { // bad_msg_seqno
			return nil, false, err
		}
		errorCode, err := r.int()
		if err != nil {
			return nil, false, err
		}
		newSalt, err := r.long()
		if err != nil {
			return nil, false, err
		}
		s.serverSalt = int64(newSalt)
		return nil, false, fmt.Errorf("mtproto: bad_server_salt error_code=%d new_salt=%d", errorCode, int64(newSalt))
	case constructorBadMsgNotification:
		badMsgID, err := r.long()
		if err != nil {
			return nil, false, err
		}
		badSeqNo, err := r.int()
		if err != nil {
			return nil, false, err
		}
		errorCode, err := r.int()
		if err != nil {
			return nil, false, err
		}
		return nil, false, fmt.Errorf("mtproto: bad_msg_notification bad_msg_id=%d seq=%d code=%d", badMsgID, badSeqNo, errorCode)
	case constructorMsgsAck:
		return nil, false, nil
	default:
		return nil, false, nil
	}
}

func parseRPCResultObject(responseMsgID int64, body []byte) (*InvokeResult, bool, error) {
	if len(body) < 4 {
		return nil, false, errors.New("mtproto: empty rpc result")
	}
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, false, err
	}
	if constructor == constructorRpcError {
		code, err := r.int()
		if err != nil {
			return nil, false, err
		}
		message, err := r.string()
		if err != nil {
			return nil, false, err
		}
		return nil, false, &RPCError{Code: int32(code), Message: cleanRPCErrorMessage(message)}
	}
	gzipPacked := false
	resultBody := body
	if constructor == constructorGzipPacked {
		packed, err := r.bytes()
		if err != nil {
			return nil, false, err
		}
		unpacked, err := gunzip(packed)
		if err != nil {
			return nil, false, err
		}
		gzipPacked = true
		resultBody = unpacked
		if len(resultBody) < 4 {
			return nil, false, errors.New("mtproto: empty gzip unpacked rpc result")
		}
		constructor = binary.LittleEndian.Uint32(resultBody[:4])
	}
	return &InvokeResult{
		ResponseMessageID: responseMsgID,
		Constructor:       constructor,
		Body:              resultBody,
		GzipPacked:        gzipPacked,
	}, true, nil
}

func mtproto2AESKeyIV(authKey []byte, msgKey []byte, x int) ([]byte, []byte) {
	sha256A := sha256Bytes(msgKey, authKey[x:x+36])
	sha256B := sha256Bytes(authKey[40+x:76+x], msgKey)
	key := make([]byte, 0, 32)
	key = append(key, sha256A[0:8]...)
	key = append(key, sha256B[8:24]...)
	key = append(key, sha256A[24:32]...)
	iv := make([]byte, 0, 32)
	iv = append(iv, sha256B[0:8]...)
	iv = append(iv, sha256A[8:24]...)
	iv = append(iv, sha256B[24:32]...)
	return key, iv
}

func makeHelpGetConfigQuery(apiID int) []byte {
	var help tlBuffer
	help.putInt(constructorHelpGetConfig)
	if apiID <= 0 {
		return help.bytes()
	}

	var init tlBuffer
	init.putInt(constructorInitConnection)
	init.putInt(0) // flags: no proxy, no params
	init.putInt(uint32(int32(apiID)))
	init.putString("MtProGo")
	init.putString("Go")
	init.putString("v10.0.0-dev")
	init.putString("en")
	init.putString("")
	init.putString("en")
	init.putRaw(help.bytes())

	var invoke tlBuffer
	invoke.putInt(constructorInvokeWithLayer)
	invoke.putInt(uint32(int32(defaultAPILayer)))
	invoke.putRaw(init.bytes())
	return invoke.bytes()
}

func gunzip(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func cleanRPCErrorMessage(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", ""))
}
