package mtproto

import (
	"crypto/rsa"
	"errors"
	"fmt"
)

const mainProductionRSAPublicKey = `-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEA6LszBcC1LGzyr992NzE0ieY+BSaOW622Aa9Bd4ZHLl+TuFQ4lo4g
5nKaMBwK/BIb9xUfg0Q29/2mgIR6Zr9krM7HjuIcCzFvDtr+L0GQjae9H0pRB2OO
62cECs5HKhT5DZ98K33vmWiLowc621dQuwKWSQKjWf50XYFw42h21P2KXUGyp2y/
+aEyZ+uVgLLQbRA1dEjSDZ2iGRy12Mk5gpYc397aYp438fsJoHIgJ2lgMv5h7WY9
t6N/byY9Nw9p21Og3AoXSL2q/2IJ1WRUhebgAdGVMlV1fkuOQoEzR7EdpqtQD9Cs
5+bfo3Nhmcyvk5ftB0WkJ9z6bNZ7yxrP8wIDAQAB
-----END RSA PUBLIC KEY-----`

// RSAPublicKey is a Telegram MTProto server RSA public key plus its MTProto fingerprint.
type RSAPublicKey struct {
	Key         *rsa.PublicKey
	Fingerprint uint64
}

// ProductionRSAPublicKeys returns the built-in Telegram production server keys used for MTProto auth.
func ProductionRSAPublicKeys() ([]RSAPublicKey, error) {
	key, err := parseRSAPublicKeyPEM(mainProductionRSAPublicKey)
	if err != nil {
		return nil, err
	}
	return []RSAPublicKey{{Key: key, Fingerprint: rsaFingerprint(key)}}, nil
}

func findRSAPublicKey(fingerprints []uint64) (*RSAPublicKey, error) {
	keys, err := ProductionRSAPublicKeys()
	if err != nil {
		return nil, err
	}
	for _, fp := range fingerprints {
		for i := range keys {
			if keys[i].Fingerprint == fp {
				return &keys[i], nil
			}
		}
	}
	return nil, fmt.Errorf("mtproto: no matching RSA public key for server fingerprints %x", fingerprints)
}

func mustProductionKey() RSAPublicKey {
	keys, err := ProductionRSAPublicKeys()
	if err != nil || len(keys) == 0 {
		panic(errors.New("mtproto: production RSA key unavailable"))
	}
	return keys[0]
}
