package keygen

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// Encoding selects the base64 flavour used to marshal keys.
type Encoding int

const (
	// RawURL is the base64.RawURLEncoding VLESS / v2fly conventionally use.
	RawURL Encoding = iota
	// Std is the standard (PEM-style) base64 WireGuard uses.
	Std
)

func (e Encoding) resolve() *base64.Encoding {
	if e == Std {
		return base64.StdEncoding
	}
	return base64.RawURLEncoding
}

// KeyPair holds a Curve25519 private/public key pair.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// Encode renders the key pair as base64 strings with the given flavour.
func (k KeyPair) Encode(enc Encoding) (priv, pub string) {
	e := enc.resolve()
	return e.EncodeToString(k.Private), e.EncodeToString(k.Public)
}

// GenerateCurve25519 produces a Curve25519 key pair. If inputBase64 is
// non-empty it is decoded with the given encoding and used as the private
// key; otherwise a random 32-byte scalar is generated and clamped to the
// Curve25519 specification.
func GenerateCurve25519(enc Encoding, inputBase64 string) (KeyPair, error) {
	priv, err := decodeOrGenerate(enc, inputBase64)
	if err != nil {
		return KeyPair{}, err
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return KeyPair{}, fmt.Errorf("derive public key: %w", err)
	}
	return KeyPair{Private: priv, Public: pub}, nil
}

func decodeOrGenerate(enc Encoding, inputBase64 string) ([]byte, error) {
	if len(inputBase64) > 0 {
		decoded, err := enc.resolve().DecodeString(inputBase64)
		if err != nil {
			return nil, fmt.Errorf("decode private key: %w", err)
		}
		if len(decoded) != curve25519.ScalarSize {
			return nil, fmt.Errorf("invalid private key length: %d (want %d)", len(decoded), curve25519.ScalarSize)
		}
		return decoded, nil
	}
	priv := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(priv); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	// Clamp as per RFC 7748 §5.
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	return priv, nil
}
