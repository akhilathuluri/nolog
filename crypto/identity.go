package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"crypto/ed25519"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/curve25519"
)

// Identity represents an ephemeral X25519 and Ed25519 identity.
type Identity struct {
	PrivateKey []byte             // X25519
	PublicKey  []byte             // X25519
	SignKey    ed25519.PrivateKey // Ed25519
	VerifyKey  ed25519.PublicKey  // Ed25519
	UniqueID   string
}

// GenerateIdentity creates a new ephemeral X25519 keypair and derives the UniqueID.
func GenerateIdentity() (*Identity, error) {
	privateKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, privateKey); err != nil {
		return nil, fmt.Errorf("failed to read random bytes for private key: %w", err)
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	// Generate Ed25519 Keypair for signing room messages
	verifyKey, signKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Hash Ed25519 verify key with SHA-256 to create the identity
	hash := sha256.Sum256(verifyKey)

	// Encode with Base58 for the Unique ID
	uniqueID := base58.Encode(hash[:])

	return &Identity{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		SignKey:    signKey,
		VerifyKey:  verifyKey,
		UniqueID:   uniqueID,
	}, nil
}

// Wipe safely zeroes out the private and public keys from memory.
func (id *Identity) Wipe() {
	if id == nil {
		return
	}
	if id.PrivateKey != nil {
		for i := range id.PrivateKey {
			id.PrivateKey[i] = 0
		}
	}
	if id.PublicKey != nil {
		for i := range id.PublicKey {
			id.PublicKey[i] = 0
		}
	}
	if id.SignKey != nil {
		for i := range id.SignKey {
			id.SignKey[i] = 0
		}
	}
	if id.VerifyKey != nil {
		for i := range id.VerifyKey {
			id.VerifyKey[i] = 0
		}
	}
	id.UniqueID = ""
}

// FingerprintPubKey returns an 8-character Base58 hash of a public key.
func FingerprintPubKey(pubKey []byte) string {
	hash := sha256.Sum256(pubKey)
	return base58.Encode(hash[:])[:8]
}

// Fingerprint returns an 8-character Base58 hash of the public key for manual verification.
func (id *Identity) Fingerprint() string {
	return FingerprintPubKey(id.PublicKey)
}
