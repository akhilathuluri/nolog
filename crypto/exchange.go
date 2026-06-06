package crypto

import (
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// DeriveSharedKey performs an X25519 Diffie-Hellman exchange to derive a symmetric key.
func DeriveSharedKey(privateKey []byte, peerPublicKey []byte) ([]byte, error) {
	sharedSecret, err := curve25519.X25519(privateKey, peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// Hash the shared secret to ensure uniform distribution
	hash := sha256.Sum256(sharedSecret)
	
	// Zero out the shared secret byte array from memory
	for i := range sharedSecret {
		sharedSecret[i] = 0
	}

	out := make([]byte, 32)
	copy(out, hash[:])
	
	// Zero out the hash array from stack memory
	for i := range hash {
		hash[i] = 0
	}

	return out, nil
}
