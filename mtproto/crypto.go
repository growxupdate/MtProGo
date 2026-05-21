package mtproto

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
)

func sha1Bytes(parts ...[]byte) []byte {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write(part)
	}
	return h.Sum(nil)
}

func sha256Bytes(parts ...[]byte) []byte {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write(part)
	}
	return h.Sum(nil)
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func reverseBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

func parseRSAPublicKeyPEM(data string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.New("rsa: pem block not found")
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("rsa: key is not RSA")
	}
	return key, nil
}

func rsaFingerprint(key *rsa.PublicKey) uint64 {
	var bare tlBuffer
	bare.putBytes(key.N.Bytes())
	bare.putBytes(big.NewInt(int64(key.E)).Bytes())
	sum := sha1.Sum(bare.bytes())
	return binary.LittleEndian.Uint64(sum[12:20])
}

func rsaPad(data []byte, key *rsa.PublicKey, rng io.Reader) ([]byte, error) {
	if len(data) > 144 {
		return nil, errors.New("mtproto: RSA_PAD data is longer than 144 bytes")
	}
	if key.Size() != 256 {
		return nil, errors.New("mtproto: expected 2048-bit RSA key")
	}

	modulus := key.N
	for attempt := 0; attempt < 32; attempt++ {
		dataWithPadding := make([]byte, 192)
		copy(dataWithPadding, data)
		if _, err := io.ReadFull(rng, dataWithPadding[len(data):]); err != nil {
			return nil, err
		}

		dataPadReversed := reverseBytes(dataWithPadding)
		tempKey := make([]byte, 32)
		if _, err := io.ReadFull(rng, tempKey); err != nil {
			return nil, err
		}

		dataWithHash := append(dataPadReversed, sha256Bytes(tempKey, dataWithPadding)...)
		aesEncrypted, err := aesIGEEncrypt(dataWithHash, tempKey, make([]byte, 32))
		if err != nil {
			return nil, err
		}
		tempKeyXOR := xorBytes(tempKey, sha256Bytes(aesEncrypted))
		keyAESEncrypted := append(tempKeyXOR, aesEncrypted...)
		if len(keyAESEncrypted) != 256 {
			return nil, errors.New("mtproto: invalid RSA_PAD length")
		}
		if new(big.Int).SetBytes(keyAESEncrypted).Cmp(modulus) >= 0 {
			continue
		}
		return rsaRawEncrypt(key, keyAESEncrypted), nil
	}
	return nil, errors.New("mtproto: RSA_PAD failed after retries")
}

func rsaRawEncrypt(key *rsa.PublicKey, plain []byte) []byte {
	m := new(big.Int).SetBytes(plain)
	e := big.NewInt(int64(key.E))
	c := new(big.Int).Exp(m, e, key.N)
	out := c.Bytes()
	if len(out) >= key.Size() {
		return out
	}
	padded := make([]byte, key.Size())
	copy(padded[key.Size()-len(out):], out)
	return padded
}

func aesIGEEncrypt(plain, key, iv []byte) ([]byte, error) {
	if len(plain)%aes.BlockSize != 0 {
		return nil, errors.New("aes-ige: plaintext length must be a multiple of 16")
	}
	if len(iv) != 32 {
		return nil, errors.New("aes-ige: iv must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(plain))
	prevC := append([]byte(nil), iv[:16]...)
	prevP := append([]byte(nil), iv[16:]...)
	buf := make([]byte, aes.BlockSize)
	enc := make([]byte, aes.BlockSize)
	for off := 0; off < len(plain); off += aes.BlockSize {
		p := plain[off : off+aes.BlockSize]
		for i := 0; i < aes.BlockSize; i++ {
			buf[i] = p[i] ^ prevC[i]
		}
		block.Encrypt(enc, buf)
		c := out[off : off+aes.BlockSize]
		for i := 0; i < aes.BlockSize; i++ {
			c[i] = enc[i] ^ prevP[i]
		}
		copy(prevC, c)
		copy(prevP, p)
	}
	return out, nil
}

func aesIGEDecrypt(ciphertext, key, iv []byte) ([]byte, error) {
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("aes-ige: ciphertext length must be a multiple of 16")
	}
	if len(iv) != 32 {
		return nil, errors.New("aes-ige: iv must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(ciphertext))
	prevC := append([]byte(nil), iv[:16]...)
	prevP := append([]byte(nil), iv[16:]...)
	buf := make([]byte, aes.BlockSize)
	dec := make([]byte, aes.BlockSize)
	for off := 0; off < len(ciphertext); off += aes.BlockSize {
		c := ciphertext[off : off+aes.BlockSize]
		for i := 0; i < aes.BlockSize; i++ {
			buf[i] = c[i] ^ prevP[i]
		}
		block.Decrypt(dec, buf)
		p := out[off : off+aes.BlockSize]
		for i := 0; i < aes.BlockSize; i++ {
			p[i] = dec[i] ^ prevC[i]
		}
		copy(prevC, c)
		copy(prevP, p)
	}
	return out, nil
}

func tempAESKeys(newNonce [32]byte, serverNonce [16]byte) (key, iv []byte) {
	nn := newNonce[:]
	sn := serverNonce[:]
	sha1A := sha1Bytes(nn, sn)
	sha1B := sha1Bytes(sn, nn)
	sha1C := sha1Bytes(nn, nn)
	key = append([]byte{}, sha1A...)
	key = append(key, sha1B[:12]...)
	iv = append([]byte{}, sha1B[12:20]...)
	iv = append(iv, sha1C...)
	iv = append(iv, nn[:4]...)
	return key, iv
}

func addSHA1AndPad16(data []byte) ([]byte, error) {
	out := append(sha1Bytes(data), data...)
	padding := 16 - (len(out) % 16)
	if padding == 16 {
		padding = 0
	}
	if padding > 0 {
		randPad, err := randomBytes(padding)
		if err != nil {
			return nil, err
		}
		out = append(out, randPad...)
	}
	return out, nil
}

func stripSHA1AndPadding(data []byte) ([]byte, error) {
	if len(data) < 20 {
		return nil, errors.New("mtproto: short SHA1 protected data")
	}
	want := data[:20]
	payloadAndPadding := data[20:]
	for n := len(payloadAndPadding); n >= 0 && len(payloadAndPadding)-n <= 15; n-- {
		candidate := payloadAndPadding[:n]
		sum := sha1Bytes(candidate)
		if equalBytes(sum, want) {
			return candidate, nil
		}
	}
	return nil, errors.New("mtproto: SHA1 check failed")
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func serverSalt(newNonce [32]byte, serverNonce [16]byte) int64 {
	var salt [8]byte
	for i := 0; i < 8; i++ {
		salt[i] = newNonce[i] ^ serverNonce[i]
	}
	return int64(binary.LittleEndian.Uint64(salt[:]))
}

func authKeyID(authKey []byte) uint64 {
	sum := sha1Bytes(authKey)
	return binary.LittleEndian.Uint64(sum[12:20])
}

func authKeyAuxHash(authKey []byte) []byte {
	sum := sha1Bytes(authKey)
	return sum[:8]
}

func newNonceHash(newNonce [32]byte, number byte, authKey []byte) [16]byte {
	aux := authKeyAuxHash(authKey)
	data := make([]byte, 0, len(newNonce)+1+len(aux))
	data = append(data, newNonce[:]...)
	data = append(data, number)
	data = append(data, aux...)
	sum := sha1Bytes(data)
	var out [16]byte
	copy(out[:], sum[4:20])
	return out
}
