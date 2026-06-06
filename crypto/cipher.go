package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// CipherEngine handles encryption and decryption using ChaCha20-Poly1305.
type CipherEngine struct {
	aead cipher.AEAD
}

// NewCipherEngine creates a new cipher engine using the provided 32-byte symmetric key.
func NewCipherEngine(key []byte) (*CipherEngine, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ChaCha20-Poly1305: %w", err)
	}
	return &CipherEngine{aead: aead}, nil
}

// Encrypt encrypts a plaintext byte slice.
func (ce *CipherEngine) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, ce.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := ce.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts a ciphertext byte slice.
func (ce *CipherEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < ce.aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:ce.aead.NonceSize()], ciphertext[ce.aead.NonceSize():]
	plaintext, err := ce.aead.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
