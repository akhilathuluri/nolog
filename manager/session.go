package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// Session represents a connected user's ephemeral session.
type Session struct {
	UniqueID    string
	UploadToken string
	
	// Channels for routing messages from/to the peer
	Incoming chan []byte
	Outgoing chan []byte
	Uploads  chan []byte
	
	// Peer ID if connected
	PeerID string
	
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSession creates a new session.
func NewSession(uniqueID string) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	tokenHex := hex.EncodeToString(tokenBytes)
	
	return &Session{
		UniqueID:    uniqueID,
		UploadToken: tokenHex,
		Incoming:    make(chan []byte, 100),
		Outgoing:    make(chan []byte, 100),
		Uploads:     make(chan []byte, 10),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Close terminates the session and its pipes.
func (s *Session) Close() {
	s.cancel()
}
