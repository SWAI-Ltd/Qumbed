package crypto

import (
	"crypto/rand"
	"io"

	"golang.org/x/crypto/nacl/box"
)

const (
	PublicKeySize  = 32
	PrivateKeySize = 32
	NonceSize      = 24
)

// KeyPair holds a Curve25519 key pair (WireGuard-style)
type KeyPair struct {
	Public  *[PublicKeySize]byte
	Private *[PrivateKeySize]byte
}

// GenerateKeyPair creates a new X25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	public, private, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{Public: public, Private: private}, nil
}

// Seal encrypts plaintext for the recipient. Overhead is box.Overhead bytes.
func Seal(plaintext []byte, recipient *[PublicKeySize]byte, sender *[PrivateKeySize]byte) ([]byte, error) {
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return box.Seal(nonce[:], plaintext, &nonce, recipient, sender), nil
}

// Open decrypts ciphertext from the sender. The nonce is prepended.
func Open(ciphertext []byte, sender *[PublicKeySize]byte, recipient *[PrivateKeySize]byte) ([]byte, bool) {
	if len(ciphertext) < NonceSize+box.Overhead {
		return nil, false
	}
	var nonce [NonceSize]byte
	copy(nonce[:], ciphertext[:NonceSize])
	return box.Open(nil, ciphertext[NonceSize:], &nonce, sender, recipient)
}

// KeyID returns first 8 bytes of public key as routing ID
func KeyID(pub *[PublicKeySize]byte) []byte {
	return pub[:8]
}
