package mtproto

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	constructorAccountGetPassword                                                = 0x548a30f5
	constructorAccountPassword                                                   = 0x957b50fb
	constructorPasswordKDFAlgoSHA256SHA256PBKDF2HMACSHA512Iter100000SHA256ModPow = 0x3a912d4a
	constructorPasswordKDFAlgoUnknown                                            = 0xd45ab096
	constructorInputCheckPasswordSRP                                             = 0xd27ff082
	constructorAuthCheckPassword                                                 = 0xd18b4d16
)

// AccountPassword contains the SRP parameters returned by account.getPassword.
type AccountPassword struct {
	Flags       uint32
	HasRecovery bool
	HasPassword bool
	Hint        string
	SRPID       int64
	SRPB        []byte
	CurrentAlgo PasswordKDFAlgo
	Raw         []byte
}

// PasswordKDFAlgo contains the current supported Telegram SRP KDF parameters.
type PasswordKDFAlgo struct {
	Constructor uint32
	Salt1       []byte
	Salt2       []byte
	G           int32
	P           []byte
}

func (a PasswordKDFAlgo) ConstructorHex() string { return fmt.Sprintf("0x%08x", a.Constructor) }

// SRPProof is the generated inputCheckPasswordSRP payload values.
type SRPProof struct {
	SRPID int64
	A     []byte
	M1    []byte
}

// AccountGetPassword calls account.getPassword and parses the account.Password result.
func (c *EncryptedClient) AccountGetPassword(ctx context.Context) (*AccountPassword, *InvokeResult, error) {
	var q tlBuffer
	q.putInt(constructorAccountGetPassword)
	result, err := c.Invoke(ctx, q.bytes())
	if err != nil {
		return nil, nil, err
	}
	password, err := parseAccountPassword(result.Body)
	if err != nil {
		return nil, result, err
	}
	result.Message = "account.getPassword succeeded over encrypted MTProto"
	return password, result, nil
}

// AuthCheckPassword completes 2FA authorization using auth.checkPassword.
func (c *EncryptedClient) AuthCheckPassword(ctx context.Context, accountPassword *AccountPassword, password string) (*InvokeResult, error) {
	if accountPassword == nil {
		return nil, errors.New("account password parameters are required")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("2FA password is required")
	}
	proof, err := BuildSRPProof(accountPassword, password)
	if err != nil {
		return nil, err
	}
	result, err := c.Invoke(ctx, makeAuthCheckPasswordQuery(proof))
	if err != nil {
		return nil, err
	}
	result.Message = "auth.checkPassword succeeded over encrypted MTProto"
	return result, nil
}

// BuildSRPProof generates the InputCheckPasswordSRP A and M1 values using Telegram's SRP 2FA algorithm.
func BuildSRPProof(passwordParams *AccountPassword, password string) (*SRPProof, error) {
	if passwordParams == nil {
		return nil, errors.New("account password parameters are required")
	}
	if !passwordParams.HasPassword {
		return nil, errors.New("account does not have a 2FA password")
	}
	algo := passwordParams.CurrentAlgo
	if algo.Constructor != constructorPasswordKDFAlgoSHA256SHA256PBKDF2HMACSHA512Iter100000SHA256ModPow {
		return nil, fmt.Errorf("unsupported password KDF algorithm 0x%08x", algo.Constructor)
	}
	if len(algo.P) == 0 || len(passwordParams.SRPB) == 0 {
		return nil, errors.New("missing SRP parameters")
	}
	if algo.G < 2 || algo.G > 7 {
		return nil, fmt.Errorf("unsupported SRP generator g=%d", algo.G)
	}

	p := new(big.Int).SetBytes(algo.P)
	g := big.NewInt(int64(algo.G))
	gb := new(big.Int).SetBytes(passwordParams.SRPB)
	if p.Sign() <= 0 || gb.Sign() <= 0 || gb.Cmp(p) >= 0 {
		return nil, errors.New("invalid SRP p or B parameter")
	}

	padSize := len(algo.P)
	pBytes := padBigIntBytes(p, padSize)
	gBytes := padBigIntBytes(g, padSize)
	gbBytes := padBigIntBytes(gb, padSize)

	k := new(big.Int).SetBytes(sha256Bytes(pBytes, gBytes))
	xBytes := passwordHashPH2([]byte(password), algo.Salt1, algo.Salt2)
	x := new(big.Int).SetBytes(xBytes)
	v := new(big.Int).Exp(g, x, p)
	kv := new(big.Int).Mul(k, v)
	kv.Mod(kv, p)

	a, err := randomBigIntLessThan(p)
	if err != nil {
		return nil, err
	}
	ga := new(big.Int).Exp(g, a, p)
	if ga.Sign() <= 0 {
		return nil, errors.New("invalid generated SRP A parameter")
	}
	gaBytes := padBigIntBytes(ga, padSize)

	u := new(big.Int).SetBytes(sha256Bytes(gaBytes, gbBytes))
	if u.Sign() == 0 {
		return nil, errors.New("invalid SRP u parameter")
	}

	t := new(big.Int).Sub(gb, kv)
	t.Mod(t, p)
	if t.Sign() <= 0 {
		return nil, errors.New("invalid SRP t parameter")
	}

	exp := new(big.Int).Mul(u, x)
	exp.Add(exp, a)
	s := new(big.Int).Exp(t, exp, p)
	ka := sha256Bytes(padBigIntBytes(s, padSize))

	hp := sha256Bytes(pBytes)
	hg := sha256Bytes(gBytes)
	hxor := xorBytes(hp, hg)
	m1 := sha256Bytes(
		hxor,
		sha256Bytes(algo.Salt1),
		sha256Bytes(algo.Salt2),
		gaBytes,
		gbBytes,
		ka,
	)

	return &SRPProof{SRPID: passwordParams.SRPID, A: gaBytes, M1: m1}, nil
}

func makeAuthCheckPasswordQuery(proof *SRPProof) []byte {
	var q tlBuffer
	q.putInt(constructorAuthCheckPassword)
	q.putInt(constructorInputCheckPasswordSRP)
	q.putLong(uint64(proof.SRPID))
	q.putBytes(proof.A)
	q.putBytes(proof.M1)
	return q.bytes()
}

func parseAccountPassword(body []byte) (*AccountPassword, error) {
	r := newTLReader(body)
	constructor, err := r.int()
	if err != nil {
		return nil, err
	}
	if constructor != constructorAccountPassword {
		return nil, fmt.Errorf("mtproto: expected account.password, got 0x%08x", constructor)
	}
	flags, err := r.int()
	if err != nil {
		return nil, err
	}
	out := &AccountPassword{
		Flags:       flags,
		HasRecovery: flags&(1<<0) != 0,
		HasPassword: flags&(1<<2) != 0,
		Raw:         append([]byte(nil), body...),
	}
	if flags&(1<<2) != 0 {
		algo, err := parsePasswordKDFAlgo(r)
		if err != nil {
			return nil, err
		}
		out.CurrentAlgo = algo
		srpB, err := r.bytes()
		if err != nil {
			return nil, err
		}
		out.SRPB = srpB
		srpID, err := r.long()
		if err != nil {
			return nil, err
		}
		out.SRPID = int64(srpID)
	}
	if flags&(1<<3) != 0 {
		hint, err := r.string()
		if err != nil {
			return nil, err
		}
		out.Hint = hint
	}
	if flags&(1<<4) != 0 {
		if _, err := r.string(); err != nil { // email_unconfirmed_pattern
			return nil, err
		}
	}
	// new_algo, new_secure_algo and secure_random are always present.
	if err := skipPasswordKDFAlgo(r); err != nil {
		return nil, err
	}
	if err := skipSecurePasswordKDFAlgo(r); err != nil {
		return nil, err
	}
	if _, err := r.bytes(); err != nil { // secure_random
		return nil, err
	}
	if flags&(1<<5) != 0 {
		if _, err := r.int(); err != nil {
			return nil, err
		}
	}
	if flags&(1<<6) != 0 {
		if _, err := r.string(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parsePasswordKDFAlgo(r *tlReader) (PasswordKDFAlgo, error) {
	constructor, err := r.int()
	if err != nil {
		return PasswordKDFAlgo{}, err
	}
	switch constructor {
	case constructorPasswordKDFAlgoUnknown:
		return PasswordKDFAlgo{Constructor: constructor}, nil
	case constructorPasswordKDFAlgoSHA256SHA256PBKDF2HMACSHA512Iter100000SHA256ModPow:
		salt1, err := r.bytes()
		if err != nil {
			return PasswordKDFAlgo{}, err
		}
		salt2, err := r.bytes()
		if err != nil {
			return PasswordKDFAlgo{}, err
		}
		g, err := r.int()
		if err != nil {
			return PasswordKDFAlgo{}, err
		}
		p, err := r.bytes()
		if err != nil {
			return PasswordKDFAlgo{}, err
		}
		return PasswordKDFAlgo{Constructor: constructor, Salt1: salt1, Salt2: salt2, G: int32(g), P: p}, nil
	default:
		return PasswordKDFAlgo{}, fmt.Errorf("unsupported password KDF algorithm 0x%08x", constructor)
	}
}

func skipPasswordKDFAlgo(r *tlReader) error {
	_, err := parsePasswordKDFAlgo(r)
	return err
}

func skipSecurePasswordKDFAlgo(r *tlReader) error {
	constructor, err := r.int()
	if err != nil {
		return err
	}
	switch constructor {
	case 0x004a8537: // securePasswordKdfAlgoUnknown#4a8537
		return nil
	case 0xbbf2dda0: // securePasswordKdfAlgoPBKDF2HMACSHA512iter100000 salt:bytes
		_, err := r.bytes()
		return err
	case 0x86471d92: // securePasswordKdfAlgoSHA512 salt:bytes
		_, err := r.bytes()
		return err
	default:
		return fmt.Errorf("unsupported secure password KDF algorithm 0x%08x", constructor)
	}
}

func passwordHashPH2(password, salt1, salt2 []byte) []byte {
	ph1 := saltedHash(saltedHash(password, salt1), salt2)
	pbkdf := pbkdf2HMACSHA512(ph1, salt1, 100000, 64)
	return saltedHash(pbkdf, salt2)
}

func saltedHash(data, salt []byte) []byte {
	return sha256Bytes(salt, data, salt)
}

func pbkdf2HMACSHA512(password, salt []byte, iter, keyLen int) []byte {
	if iter <= 0 || keyLen <= 0 {
		return nil
	}
	hLen := sha512.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	var intBlock [4]byte
	for block := 1; block <= numBlocks; block++ {
		intBlock[0] = byte(block >> 24)
		intBlock[1] = byte(block >> 16)
		intBlock[2] = byte(block >> 8)
		intBlock[3] = byte(block)

		mac := hmac.New(sha512.New, password)
		mac.Write(salt)
		mac.Write(intBlock[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)

		for i := 1; i < iter; i++ {
			mac = hmac.New(sha512.New, password)
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

func randomBigIntLessThan(max *big.Int) (*big.Int, error) {
	if max == nil || max.Sign() <= 0 {
		return nil, errors.New("invalid max random integer")
	}
	for {
		b := make([]byte, len(max.Bytes()))
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		n := new(big.Int).SetBytes(b)
		if n.Sign() > 0 && n.Cmp(max) < 0 {
			return n, nil
		}
	}
}
