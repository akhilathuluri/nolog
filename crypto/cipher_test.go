package crypto

import (
	"bytes"
	"testing"
)

func TestCipherEngine_ReplayAttack(t *testing.T) {
	engine, err := NewCipherEngine(make([]byte, 32), true, true)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	plaintext := []byte("Hello Replay Test")
	ciphertext, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// First decryption should succeed
	decrypted, err := engine.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("First decryption failed: %v", err)
	}
	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Decrypted text mismatch")
	}

	// Second decryption of exact same ciphertext should fail (Replay Attack)
	_, err = engine.Decrypt(ciphertext)
	if err == nil {
		t.Fatalf("Expected replay attack to be blocked, but it succeeded!")
	}
	if err.Error() != "message rejected: replay attack detected (duplicate)" {
		t.Fatalf("Expected replay attack error, got: %v", err)
	}
}

func TestCipherEngine_SplitKeysRatcheting(t *testing.T) {
	sharedSecret := make([]byte, 32)
	for i := range sharedSecret {
		sharedSecret[i] = byte(i)
	}

	// Alice is initiator, Bob is responder
	alice, _ := NewCipherEngine(sharedSecret, true, false)
	bob, _ := NewCipherEngine(sharedSecret, false, false)

	// Test bidirectional communication
	msgA1 := []byte("Alice to Bob 1")
	msgB1 := []byte("Bob to Alice 1")
	msgA2 := []byte("Alice to Bob 2")

	// Alice encrypts two messages concurrently (Bob hasn't received anything yet)
	ciphA1, _ := alice.Encrypt(msgA1)
	ciphA2, _ := alice.Encrypt(msgA2)

	// Bob encrypts a message concurrently
	ciphB1, _ := bob.Encrypt(msgB1)

	// Alice decrypts Bob's message
	decB1, err := alice.Decrypt(ciphB1)
	if err != nil || !bytes.Equal(decB1, msgB1) {
		t.Fatalf("Alice failed to decrypt Bob's message: %v", err)
	}

	// Bob decrypts Alice's messages
	decA1, err := bob.Decrypt(ciphA1)
	if err != nil || !bytes.Equal(decA1, msgA1) {
		t.Fatalf("Bob failed to decrypt Alice's message 1: %v", err)
	}

	decA2, err := bob.Decrypt(ciphA2)
	if err != nil || !bytes.Equal(decA2, msgA2) {
		t.Fatalf("Bob failed to decrypt Alice's message 2: %v", err)
	}
}
