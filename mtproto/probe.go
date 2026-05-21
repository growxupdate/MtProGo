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
	"sort"
	"time"
)

// DefaultDCs are public Telegram production DC addresses.
var DefaultDCs = []string{
	"149.154.167.50:443",
	"149.154.167.51:443",
	"149.154.175.100:443",
	"149.154.167.91:443",
	"149.154.171.5:443",
}

// ProbeResult is the result of the first MTProto auth step.
type ProbeResult struct {
	Address         string
	Nonce           [16]byte
	ServerNonce     [16]byte
	PQ              []byte
	P               []byte
	Q               []byte
	RSAFingerprints []uint64
}

// PQHex returns pq in hex.
func (r ProbeResult) PQHex() string { return hex.EncodeToString(r.PQ) }

// PHex returns p in hex.
func (r ProbeResult) PHex() string { return hex.EncodeToString(r.P) }

// QHex returns q in hex.
func (r ProbeResult) QHex() string { return hex.EncodeToString(r.Q) }

// ProbeDefault tries Telegram default DCs until one replies to req_pq_multi.
func ProbeDefault(ctx context.Context) (*ProbeResult, error) {
	var last error
	for _, addr := range DefaultDCs {
		result, err := Probe(ctx, addr)
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

// Probe performs a pure Go raw MTProto req_pq_multi request over TCP abridged.
func Probe(ctx context.Context, address string) (*ProbeResult, error) {
	if address == "" {
		return nil, errors.New("address is required")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	}

	if err := abridgedInit(conn); err != nil {
		return nil, err
	}

	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	var request tlBuffer
	request.putInt(constructorReqPQMulti)
	request.putRaw(nonce[:])

	msgID := nextMessageID()
	packet := buildUnencryptedMessage(msgID, request.bytes())
	if err := abridgedWrite(conn, packet); err != nil {
		return nil, err
	}

	payload, err := abridgedRead(conn)
	if err != nil {
		return nil, err
	}
	body, err := parseUnencryptedMessage(payload)
	if err != nil {
		return nil, err
	}

	result, err := parseResPQ(body)
	if err != nil {
		return nil, err
	}
	if result.Nonce != nonce {
		return nil, errors.New("mtproto: nonce mismatch")
	}
	result.Address = address
	return result, nil
}

func nextMessageID() int64 {
	// Telegram message IDs are time-based and divisible by 4 for client messages.
	now := time.Now().UnixNano()
	seconds := now / int64(time.Second)
	nanos := now % int64(time.Second)
	msgID := (seconds << 32) | ((nanos << 32) / int64(time.Second))
	msgID &= ^int64(3)
	return msgID
}

func parseResPQ(body []byte) (*ProbeResult, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	if constructor != constructorResPQ {
		return nil, fmt.Errorf("mtproto: expected resPQ constructor, got 0x%08x", constructor)
	}

	var result ProbeResult
	nonce, err := r.raw(16)
	if err != nil {
		return nil, err
	}
	copy(result.Nonce[:], nonce)
	serverNonce, err := r.raw(16)
	if err != nil {
		return nil, err
	}
	copy(result.ServerNonce[:], serverNonce)

	pq, err := r.bytes()
	if err != nil {
		return nil, err
	}
	result.PQ = pq

	vectorConstructor, err := r.int()
	if err != nil {
		return nil, err
	}
	if vectorConstructor != constructorVector {
		return nil, fmt.Errorf("mtproto: expected vector constructor, got 0x%08x", vectorConstructor)
	}
	count, err := r.int()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(count); i++ {
		fingerprint, err := r.long()
		if err != nil {
			return nil, err
		}
		result.RSAFingerprints = append(result.RSAFingerprints, fingerprint)
	}

	p, q := factorPQ(pq)
	result.P = p
	result.Q = q
	return &result, nil
}

func factorPQ(pqBytes []byte) ([]byte, []byte) {
	n := new(big.Int).SetBytes(pqBytes)
	if n.Sign() <= 0 {
		return nil, nil
	}
	factor := pollardRho(n)
	if factor == nil || factor.Sign() == 0 || factor.Cmp(n) == 0 {
		return nil, nil
	}
	other := new(big.Int).Div(new(big.Int).Set(n), factor)
	factors := []*big.Int{factor, other}
	sort.Slice(factors, func(i, j int) bool { return factors[i].Cmp(factors[j]) < 0 })
	return factors[0].Bytes(), factors[1].Bytes()
}

func pollardRho(n *big.Int) *big.Int {
	if n.ProbablyPrime(20) {
		return n
	}
	two := big.NewInt(2)
	if new(big.Int).Mod(n, two).Sign() == 0 {
		return two
	}
	one := big.NewInt(1)

	for c := int64(1); c < 20; c++ {
		x := big.NewInt(2)
		y := big.NewInt(2)
		d := big.NewInt(1)
		cc := big.NewInt(c)
		for d.Cmp(one) == 0 {
			x = rhoStep(x, cc, n)
			y = rhoStep(rhoStep(y, cc, n), cc, n)
			d.Sub(x, y)
			d.Abs(d)
			d.GCD(nil, nil, d, n)
		}
		if d.Cmp(n) != 0 {
			return new(big.Int).Set(d)
		}
	}
	return nil
}

func rhoStep(x, c, n *big.Int) *big.Int {
	z := new(big.Int).Mul(x, x)
	z.Add(z, c)
	z.Mod(z, n)
	return z
}

// Uint64FromBytes decodes a big-endian integer of up to eight bytes.
func Uint64FromBytes(b []byte) uint64 {
	var padded [8]byte
	copy(padded[8-len(b):], b)
	return binary.BigEndian.Uint64(padded[:])
}
