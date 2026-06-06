package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// CipherEngine handles encryption and decryption using ChaCha20-Poly1305.
type CipherEngine struct {
	key  []byte
	aead cipher.AEAD
}

// NewCipherEngine creates a new cipher engine using the provided 32-byte symmetric key.
func NewCipherEngine(key []byte) (*CipherEngine, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ChaCha20-Poly1305: %w", err)
	}
	return &CipherEngine{key: key, aead: aead}, nil
}

// RatchetKey rotates the symmetric key to provide forward secrecy.
func (ce *CipherEngine) RatchetKey() {
	hash := sha256.New
	kdf := hkdf.New(hash, ce.key, nil, []byte("ratchet"))
	newKey := make([]byte, 32)
	io.ReadFull(kdf, newKey)
	ce.key = newKey
	ce.aead, _ = chacha20poly1305.New(ce.key)
}

// Encrypt encrypts a plaintext byte slice.
func (ce *CipherEngine) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, ce.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Use timestamp as AAD to prevent replay attacks
	aad := make([]byte, 8)
	binary.BigEndian.PutUint64(aad, uint64(time.Now().UnixMilli()))

	ciphertext := ce.aead.Seal(nonce, nonce, plaintext, aad)
	out := append(aad, ciphertext...)
	return out, nil
}

// Decrypt decrypts a ciphertext byte slice.
func (ce *CipherEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 8+ce.aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	aad := ciphertext[:8]
	nonce := ciphertext[8 : 8+ce.aead.NonceSize()]
	actualCiphertext := ciphertext[8+ce.aead.NonceSize():]

	plaintext, err := ce.aead.Open(nil, nonce, actualCiphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	ts := binary.BigEndian.Uint64(aad)
	now := uint64(time.Now().UnixMilli())
	if now > ts && (now-ts) > 300000 {
		return nil, fmt.Errorf("message rejected: replay attack detected (too old)")
	}

	return plaintext, nil
}
