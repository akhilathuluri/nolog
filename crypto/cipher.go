package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// CipherEngine handles encryption and decryption using ChaCha20-Poly1305.
type CipherEngine struct {
	txKey        []byte
	rxKey        []byte
	txAead       cipher.AEAD
	rxAead       cipher.AEAD
	seenMessages map[string]int64
	mu           sync.Mutex
	isRoom       bool
}

// NewCipherEngine creates a new cipher engine using the provided 32-byte symmetric key.
func NewCipherEngine(sharedSecret []byte, isInitiator bool, isRoom bool) (*CipherEngine, error) {
	if isRoom {
		aead, err := chacha20poly1305.NewX(sharedSecret)
		if err != nil {
			return nil, err
		}
		return &CipherEngine{
			txKey:        sharedSecret,
			rxKey:        sharedSecret,
			txAead:       aead,
			rxAead:       aead,
			seenMessages: make(map[string]int64),
			isRoom:       true,
		}, nil
	}

	hash := sha256.New
	kdf := hkdf.New(hash, sharedSecret, nil, []byte("split-keys"))
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	io.ReadFull(kdf, key1)
	io.ReadFull(kdf, key2)

	var tx, rx []byte
	if isInitiator {
		tx, rx = key1, key2
	} else {
		tx, rx = key2, key1
	}

	txAead, _ := chacha20poly1305.NewX(tx)
	rxAead, _ := chacha20poly1305.NewX(rx)

	return &CipherEngine{
		txKey:        tx,
		rxKey:        rx,
		txAead:       txAead,
		rxAead:       rxAead,
		seenMessages: make(map[string]int64),
		isRoom:       false,
	}, nil
}

// RatchetTx rotates the transmission key.
func (ce *CipherEngine) RatchetTx() {
	hash := sha256.New
	kdf := hkdf.New(hash, ce.txKey, nil, []byte("ratchet"))
	newKey := make([]byte, 32)
	io.ReadFull(kdf, newKey)
	for i := range ce.txKey { ce.txKey[i] = 0 }
	ce.txKey = newKey
	ce.txAead, _ = chacha20poly1305.NewX(ce.txKey)
}

// RatchetRx rotates the receiving key.
func (ce *CipherEngine) RatchetRx() {
	hash := sha256.New
	kdf := hkdf.New(hash, ce.rxKey, nil, []byte("ratchet"))
	newKey := make([]byte, 32)
	io.ReadFull(kdf, newKey)
	for i := range ce.rxKey { ce.rxKey[i] = 0 }
	ce.rxKey = newKey
	ce.rxAead, _ = chacha20poly1305.NewX(ce.rxKey)
}

// Rekey performs a Diffie-Hellman ratchet by regenerating the split keys from a new shared secret.
func (ce *CipherEngine) Rekey(sharedSecret []byte, isInitiator bool, isRoom bool) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if isRoom {
		ce.txKey = sharedSecret
		ce.rxKey = sharedSecret
		ce.txAead, _ = chacha20poly1305.NewX(sharedSecret)
		ce.rxAead, _ = chacha20poly1305.NewX(sharedSecret)
		return
	}

	hash := sha256.New
	kdf := hkdf.New(hash, sharedSecret, nil, []byte("split-keys"))
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	io.ReadFull(kdf, key1)
	io.ReadFull(kdf, key2)

	if isInitiator {
		ce.txKey, ce.rxKey = key1, key2
	} else {
		ce.txKey, ce.rxKey = key2, key1
	}

	ce.txAead, _ = chacha20poly1305.NewX(ce.txKey)
	ce.rxAead, _ = chacha20poly1305.NewX(ce.rxKey)
}

// Encrypt encrypts a plaintext byte slice.
func (ce *CipherEngine) Encrypt(plaintext []byte) ([]byte, error) {
	ce.mu.Lock()
	aead := ce.txAead
	ce.mu.Unlock()

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Use timestamp as AAD to prevent replay attacks
	aad := make([]byte, 8)
	binary.BigEndian.PutUint64(aad, uint64(time.Now().UnixMilli()))

	ciphertext := aead.Seal(nonce, nonce, plaintext, aad)
	out := append(aad, ciphertext...)
	
	ce.mu.Lock()
	if !ce.isRoom {
		ce.RatchetTx()
	}
	ce.mu.Unlock()
	
	return out, nil
}

// Decrypt decrypts a ciphertext byte slice.
func (ce *CipherEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	ce.mu.Lock()
	aead := ce.rxAead
	ce.mu.Unlock()

	if len(ciphertext) < 8+aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Replay cache check
	hash := sha256.Sum256(ciphertext)
	hashStr := string(hash[:])
	
	ce.mu.Lock()
	if _, exists := ce.seenMessages[hashStr]; exists {
		ce.mu.Unlock()
		return nil, fmt.Errorf("message rejected: replay attack detected (duplicate)")
	}
	
	nowMillis := int64(time.Now().UnixMilli())
	ce.seenMessages[hashStr] = nowMillis
	
	// Lightweight garbage collection for the replay cache
	if len(ce.seenMessages) > 1000 {
		for k, v := range ce.seenMessages {
			if nowMillis-v > 300000 {
				delete(ce.seenMessages, k)
			}
		}
	}
	ce.mu.Unlock()

	aad := ciphertext[:8]
	nonce := ciphertext[8 : 8+aead.NonceSize()]
	actualCiphertext := ciphertext[8+aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, actualCiphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	ts := binary.BigEndian.Uint64(aad)
	now := uint64(time.Now().UnixMilli())
	if now > ts && (now-ts) > 300000 {
		return nil, fmt.Errorf("message rejected: replay attack detected (too old)")
	}

	ce.mu.Lock()
	if !ce.isRoom {
		ce.RatchetRx()
	}
	ce.mu.Unlock()

	return plaintext, nil
}
