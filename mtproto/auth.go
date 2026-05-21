package mtproto

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"
)

// DCOption describes a Telegram production DC TCP endpoint.
type DCOption struct {
	ID      int
	Address string
}

// DefaultDCOptions are production Telegram DC endpoints used by the examples.
var DefaultDCOptions = []DCOption{
	{ID: 1, Address: "149.154.175.53:443"},
	{ID: 1, Address: "149.154.175.50:443"},
	{ID: 2, Address: "149.154.167.51:443"},
	{ID: 2, Address: "149.154.167.50:443"},
	{ID: 3, Address: "149.154.175.100:443"},
	{ID: 4, Address: "149.154.167.91:443"},
	{ID: 5, Address: "91.108.56.130:443"},
	{ID: 5, Address: "149.154.171.5:443"},
}

// AuthKeyResult is the result of MTProto authorization key generation.
type AuthKeyResult struct {
	DCID           int
	Address        string
	Nonce          [16]byte
	ServerNonce    [16]byte
	NewNonce       [32]byte
	PQ             []byte
	P              []byte
	Q              []byte
	RSAFingerprint uint64
	G              int32
	DHPrime        []byte
	GA             []byte
	GB             []byte
	AuthKey        []byte
	AuthKeyID      uint64
	ServerSalt     int64
	ServerTime     int32
	TimeOffset     int64
	Message        string
}

// AuthKeyHex returns the auth key in hexadecimal. It is useful for local debugging only.
func (r AuthKeyResult) AuthKeyHex() string { return hex.EncodeToString(r.AuthKey) }

// AuthKeyIDHex returns the MTProto auth_key_id in hexadecimal.
func (r AuthKeyResult) AuthKeyIDHex() string {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], r.AuthKeyID)
	return hex.EncodeToString(b[:])
}

// AuthKeyDefault tries production DCs until auth key generation succeeds.
func AuthKeyDefault(ctx context.Context) (*AuthKeyResult, error) {
	var last error
	for _, dc := range DefaultDCOptions {
		result, err := AuthKey(ctx, dc)
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

// AuthKey creates an MTProto authorization key against a Telegram DC over TCP abridged.
func AuthKey(ctx context.Context, dc DCOption) (*AuthKeyResult, error) {
	conn, err := dialMTProto(ctx, dc, 25*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return authKeyOnConn(conn, dc)
}

func dialMTProto(ctx context.Context, dc DCOption, fallbackTimeout time.Duration) (net.Conn, error) {
	if dc.ID == 0 {
		return nil, errors.New("mtproto: dc id is required")
	}
	if dc.Address == "" {
		return nil, errors.New("mtproto: dc address is required")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", dc.Address)
	if err != nil {
		return nil, err
	}
	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetDeadline(deadline)
	} else if fallbackTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(fallbackTimeout))
	}
	if err := abridgedInit(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func authKeyOnConn(conn net.Conn, dc DCOption) (*AuthKeyResult, error) {
	result := &AuthKeyResult{DCID: dc.ID, Address: dc.Address}
	if _, err := rand.Read(result.Nonce[:]); err != nil {
		return nil, err
	}
	if _, err := rand.Read(result.NewNonce[:]); err != nil {
		return nil, err
	}

	resPQ, err := sendReqPQ(conn, result.Nonce)
	if err != nil {
		return nil, err
	}
	if resPQ.Nonce != result.Nonce {
		return nil, errors.New("mtproto: nonce mismatch in resPQ")
	}
	result.ServerNonce = resPQ.ServerNonce
	result.PQ = resPQ.PQ
	result.P = resPQ.P
	result.Q = resPQ.Q

	rsaKey, err := findRSAPublicKey(resPQ.RSAFingerprints)
	if err != nil {
		return nil, err
	}
	result.RSAFingerprint = rsaKey.Fingerprint

	serverDH, err := sendReqDHParams(conn, dc.ID, result, rsaKey)
	if err != nil {
		return nil, err
	}
	result.G = int32(serverDH.G)
	result.DHPrime = serverDH.DHPrime
	result.GA = serverDH.GA
	result.ServerTime = int32(serverDH.ServerTime)
	result.TimeOffset = int64(serverDH.ServerTime) - time.Now().Unix()

	if err := validateDHParams(serverDH.G, serverDH.DHPrime, serverDH.GA); err != nil {
		return nil, err
	}

	authKey, gb, err := sendSetClientDHParams(conn, result, serverDH)
	if err != nil {
		return nil, err
	}
	result.AuthKey = authKey
	result.GB = gb
	result.AuthKeyID = authKeyID(authKey)
	result.ServerSalt = serverSalt(result.NewNonce, result.ServerNonce)
	result.Message = "auth key generated"
	return result, nil
}

type resPQData struct {
	Nonce           [16]byte
	ServerNonce     [16]byte
	PQ              []byte
	P               []byte
	Q               []byte
	RSAFingerprints []uint64
}

func sendReqPQ(conn net.Conn, nonce [16]byte) (*resPQData, error) {
	var request tlBuffer
	request.putInt(constructorReqPQMulti)
	request.putRaw(nonce[:])
	body, err := sendUnencrypted(conn, request.bytes())
	if err != nil {
		return nil, err
	}
	probeResult, err := parseResPQ(body)
	if err != nil {
		return nil, err
	}
	return &resPQData{
		Nonce:           probeResult.Nonce,
		ServerNonce:     probeResult.ServerNonce,
		PQ:              probeResult.PQ,
		P:               probeResult.P,
		Q:               probeResult.Q,
		RSAFingerprints: probeResult.RSAFingerprints,
	}, nil
}

func sendUnencrypted(conn net.Conn, body []byte) ([]byte, error) {
	packet := buildUnencryptedMessage(nextMessageID(), body)
	if err := abridgedWrite(conn, packet); err != nil {
		return nil, err
	}
	payload, err := abridgedRead(conn)
	if err != nil {
		return nil, err
	}
	return parseUnencryptedMessage(payload)
}

func sendReqDHParams(conn net.Conn, dcID int, result *AuthKeyResult, key *RSAPublicKey) (*serverDHInnerData, error) {
	var inner tlBuffer
	inner.putInt(constructorPQInnerDataDC)
	inner.putBytes(result.PQ)
	inner.putBytes(result.P)
	inner.putBytes(result.Q)
	inner.putRaw(result.Nonce[:])
	inner.putRaw(result.ServerNonce[:])
	inner.putRaw(result.NewNonce[:])
	inner.putInt(uint32(int32(dcID)))

	encryptedData, err := rsaPad(inner.bytes(), key.Key, rand.Reader)
	if err != nil {
		return nil, err
	}

	var request tlBuffer
	request.putInt(constructorReqDHParams)
	request.putRaw(result.Nonce[:])
	request.putRaw(result.ServerNonce[:])
	request.putBytes(result.P)
	request.putBytes(result.Q)
	request.putLong(key.Fingerprint)
	request.putBytes(encryptedData)

	body, err := sendUnencrypted(conn, request.bytes())
	if err != nil {
		return nil, err
	}
	return parseServerDHParams(body, result)
}

type serverDHInnerData struct {
	Nonce       [16]byte
	ServerNonce [16]byte
	G           int32
	DHPrime     []byte
	GA          []byte
	ServerTime  int32
}

func parseServerDHParams(body []byte, result *AuthKeyResult) (*serverDHInnerData, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	switch constructor {
	case constructorServerDHParamsBad:
		return nil, errors.New("mtproto: server_DH_params_fail")
	case constructorServerDHParamsOK:
		// continue
	default:
		return nil, fmt.Errorf("mtproto: expected server_DH_params_ok, got 0x%08x", constructor)
	}
	nonce, err := r.raw(16)
	if err != nil {
		return nil, err
	}
	if !equalBytes(nonce, result.Nonce[:]) {
		return nil, errors.New("mtproto: nonce mismatch in server_DH_params_ok")
	}
	serverNonce, err := r.raw(16)
	if err != nil {
		return nil, err
	}
	if !equalBytes(serverNonce, result.ServerNonce[:]) {
		return nil, errors.New("mtproto: server_nonce mismatch in server_DH_params_ok")
	}
	encryptedAnswer, err := r.bytes()
	if err != nil {
		return nil, err
	}
	key, iv := tempAESKeys(result.NewNonce, result.ServerNonce)
	answerWithHash, err := aesIGEDecrypt(encryptedAnswer, key, iv)
	if err != nil {
		return nil, err
	}
	answer, err := stripSHA1AndPadding(answerWithHash)
	if err != nil {
		return nil, err
	}
	inner := newTLReader(answer)
	innerConstructor, err := inner.int()
	if err != nil {
		return nil, err
	}
	if innerConstructor != constructorServerDHInnerData {
		return nil, fmt.Errorf("mtproto: expected server_DH_inner_data, got 0x%08x", innerConstructor)
	}
	var out serverDHInnerData
	nonce, err = inner.raw(16)
	if err != nil {
		return nil, err
	}
	copy(out.Nonce[:], nonce)
	if !equalBytes(out.Nonce[:], result.Nonce[:]) {
		return nil, errors.New("mtproto: nonce mismatch in server_DH_inner_data")
	}
	serverNonce, err = inner.raw(16)
	if err != nil {
		return nil, err
	}
	copy(out.ServerNonce[:], serverNonce)
	if !equalBytes(out.ServerNonce[:], result.ServerNonce[:]) {
		return nil, errors.New("mtproto: server_nonce mismatch in server_DH_inner_data")
	}
	g, err := inner.int()
	if err != nil {
		return nil, err
	}
	out.G = int32(g)
	out.DHPrime, err = inner.bytes()
	if err != nil {
		return nil, err
	}
	out.GA, err = inner.bytes()
	if err != nil {
		return nil, err
	}
	serverTime, err := inner.int()
	if err != nil {
		return nil, err
	}
	out.ServerTime = int32(serverTime)
	return &out, nil
}

func validateDHParams(g int32, dhPrimeBytes, gaBytes []byte) error {
	if g < 2 || g > 7 {
		return fmt.Errorf("mtproto: unsupported generator %d", g)
	}
	dhPrime := new(big.Int).SetBytes(dhPrimeBytes)
	ga := new(big.Int).SetBytes(gaBytes)
	one := big.NewInt(1)
	if dhPrime.Sign() <= 0 || ga.Sign() <= 0 {
		return errors.New("mtproto: invalid DH values")
	}
	if ga.Cmp(one) <= 0 || ga.Cmp(new(big.Int).Sub(dhPrime, one)) >= 0 {
		return errors.New("mtproto: g_a outside valid range")
	}
	// Full safe-prime verification is intentionally kept out of the hot path.
	// Telegram normally sends the well-known 2048-bit safe prime. The next versions will add a cached verifier.
	if dhPrime.BitLen() < 2040 {
		return errors.New("mtproto: dh_prime is too small")
	}
	return nil
}

func sendSetClientDHParams(conn net.Conn, result *AuthKeyResult, serverDH *serverDHInnerData) ([]byte, []byte, error) {
	dhPrime := new(big.Int).SetBytes(serverDH.DHPrime)
	ga := new(big.Int).SetBytes(serverDH.GA)

	bBytes := make([]byte, 256)
	if _, err := rand.Read(bBytes); err != nil {
		return nil, nil, err
	}
	b := new(big.Int).SetBytes(bBytes)
	g := big.NewInt(int64(serverDH.G))
	gbInt := new(big.Int).Exp(g, b, dhPrime)
	gb := padBigIntBytes(gbInt, len(serverDH.DHPrime))
	authKeyInt := new(big.Int).Exp(ga, b, dhPrime)
	authKey := padBigIntBytes(authKeyInt, len(serverDH.DHPrime))

	var clientInner tlBuffer
	clientInner.putInt(constructorClientDHInnerData)
	clientInner.putRaw(result.Nonce[:])
	clientInner.putRaw(result.ServerNonce[:])
	clientInner.putLong(0)
	clientInner.putBytes(gb)

	dataWithHash, err := addSHA1AndPad16(clientInner.bytes())
	if err != nil {
		return nil, nil, err
	}
	key, iv := tempAESKeys(result.NewNonce, result.ServerNonce)
	encrypted, err := aesIGEEncrypt(dataWithHash, key, iv)
	if err != nil {
		return nil, nil, err
	}

	var request tlBuffer
	request.putInt(constructorSetClientDHParams)
	request.putRaw(result.Nonce[:])
	request.putRaw(result.ServerNonce[:])
	request.putBytes(encrypted)

	body, err := sendUnencrypted(conn, request.bytes())
	if err != nil {
		return nil, nil, err
	}
	if err := parseDHGenAnswer(body, result, authKey); err != nil {
		return nil, nil, err
	}
	return authKey, gb, nil
}

func parseDHGenAnswer(body []byte, result *AuthKeyResult, authKey []byte) error {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return err
	}
	nonce, err := r.raw(16)
	if err != nil {
		return err
	}
	if !equalBytes(nonce, result.Nonce[:]) {
		return errors.New("mtproto: nonce mismatch in dh_gen response")
	}
	serverNonce, err := r.raw(16)
	if err != nil {
		return err
	}
	if !equalBytes(serverNonce, result.ServerNonce[:]) {
		return errors.New("mtproto: server_nonce mismatch in dh_gen response")
	}
	hashBytes, err := r.raw(16)
	if err != nil {
		return err
	}
	switch constructor {
	case constructorDHGenOK:
		want := newNonceHash(result.NewNonce, 1, authKey)
		if !equalBytes(hashBytes, want[:]) {
			return errors.New("mtproto: new_nonce_hash1 mismatch")
		}
		return nil
	case constructorDHGenRetry:
		want := newNonceHash(result.NewNonce, 2, authKey)
		if !equalBytes(hashBytes, want[:]) {
			return errors.New("mtproto: new_nonce_hash2 mismatch")
		}
		return errors.New("mtproto: dh_gen_retry")
	case constructorDHGenFail:
		want := newNonceHash(result.NewNonce, 3, authKey)
		if !equalBytes(hashBytes, want[:]) {
			return errors.New("mtproto: new_nonce_hash3 mismatch")
		}
		return errors.New("mtproto: dh_gen_fail")
	default:
		return fmt.Errorf("mtproto: unknown dh_gen response 0x%08x", constructor)
	}
}

func padBigIntBytes(v *big.Int, size int) []byte {
	b := v.Bytes()
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}
