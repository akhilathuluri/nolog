package manager

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	MaxSessions        = 1000
	MaxTotalMemory     = 250 * 1024 * 1024 // 250 MB
	MaxFilesPerSession = 5
)

type Room struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewRoom() *Room {
	return &Room{
		sessions: make(map[string]*Session),
	}
}

type FileEntry struct {
	Data       []byte
	UploaderID string
}

// Hub manages active sessions securely in memory.
type Hub struct {
	mu             sync.RWMutex
	sessions       map[string]*Session
	files          map[string]*FileEntry
	rooms          map[string]*Room
	totalMemory    int64
	sessionFiles   map[string]int
}

func NewHub() *Hub {
	return &Hub{
		sessions:     make(map[string]*Session),
		files:        make(map[string]*FileEntry),
		rooms:        make(map[string]*Room),
		sessionFiles: make(map[string]int),
	}
}

// Register adds a new session to the hub.
func (h *Hub) Register(session *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[session.UniqueID] = session
}

// Unregister removes a session.
func (h *Hub) Unregister(uniqueID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[uniqueID]; ok {
		s.Close()
		delete(h.sessions, uniqueID)
	}
	delete(h.sessionFiles, uniqueID)
	
	for _, room := range h.rooms {
		room.mu.Lock()
		delete(room.sessions, uniqueID)
		room.mu.Unlock()
	}
}

// Get retrieves a session by ID.
func (h *Hub) Get(uniqueID string) (*Session, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.sessions[uniqueID]
	return s, ok
}

// Pair links two sessions together.
func (h *Hub) Pair(sourceID, targetID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	source, ok1 := h.sessions[sourceID]
	target, ok2 := h.sessions[targetID]

	if !ok1 || !ok2 {
		return fmt.Errorf("one or both sessions not found")
	}

	if source.PeerID != "" || target.PeerID != "" {
		return fmt.Errorf("one or both sessions are already paired")
	}

	source.PeerID = targetID
	target.PeerID = sourceID

	// Link them: source outgoing goes to target incoming, target outgoing goes to source incoming
	go h.pipe(source.Outgoing, target.Incoming, source.ctx, target.ctx)
	go h.pipe(target.Outgoing, source.Incoming, target.ctx, source.ctx)

	return nil
}

func (h *Hub) pipe(out chan []byte, in chan []byte, ctx1, ctx2 context.Context) {
	defer func() {
		select {
		case <-ctx2.Done():
		default:
			select {
			case in <- []byte("SYS:DISCONNECT"):
			default:
			}
		}
	}()

	for {
		select {
		case <-ctx1.Done():
			return
		case <-ctx2.Done():
			return
		case msg, ok := <-out:
			if !ok {
				return
			}
			select {
			case <-ctx1.Done():
				return
			case <-ctx2.Done():
				return
			case in <- msg:
			case <-time.After(2 * time.Second):
				// Peer is not reading messages (frozen or malicious). Drop connection.
				return
			}
		}
	}
}

// TotalMemoryUsed returns the total memory used by files.
func (h *Hub) TotalMemoryUsed() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.totalMemory
}

// GetFilesCount returns the number of files uploaded by a session.
func (h *Hub) GetFilesCount(uniqueID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessionFiles[uniqueID]
}

// StoreFile saves an ephemeral file payload in memory.
func (h *Hub) StoreFile(key string, data []byte, uploaderID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	size := int64(len(data))
	if h.totalMemory+size > MaxTotalMemory {
		return fmt.Errorf("server storage quota exceeded")
	}

	if h.sessionFiles[uploaderID] >= MaxFilesPerSession {
		return fmt.Errorf("session file limit reached")
	}

	h.files[key] = &FileEntry{Data: data, UploaderID: uploaderID}
	h.totalMemory += size
	h.sessionFiles[uploaderID]++

	// Automatically garbage collect the file after 10 minutes if not downloaded
	time.AfterFunc(10*time.Minute, func() {
		h.DeleteFile(key)
	})
	return nil
}

// GetFile retrieves an ephemeral file payload from memory.
func (h *Hub) GetFile(key string) ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.files[key]
	if !ok {
		return nil, false
	}
	return entry.Data, true
}

// DeleteFile removes a file from memory.
func (h *Hub) DeleteFile(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry, ok := h.files[key]; ok {
		size := int64(len(entry.Data))
		h.totalMemory -= size
		if h.sessionFiles[entry.UploaderID] > 0 {
			h.sessionFiles[entry.UploaderID]--
		}
		delete(h.files, key)
	}
}

// GetByUploadToken retrieves a session by its UploadToken.
func (h *Hub) GetByUploadToken(token string) (*Session, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, s := range h.sessions {
		if s.UploadToken == token {
			return s, true
		}
	}
	return nil, false
}

// SessionCount returns the number of active sessions.
func (h *Hub) SessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// CreateRoom explicitly registers a room with a 10-minute expiration.
func (h *Hub) CreateRoom(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms[roomID] = NewRoom()

	time.AfterFunc(10*time.Minute, func() {
		h.mu.Lock()
		delete(h.rooms, roomID)
		h.mu.Unlock()
	})
}

func (h *Hub) JoinRoom(roomID, sessionID string) error {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	session, ok2 := h.sessions[sessionID]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("room expired or does not exist")
	}

	if !ok2 {
		return fmt.Errorf("session not found")
	}

	room.mu.Lock()
	defer room.mu.Unlock()
	room.sessions[sessionID] = session
	return nil
}

func (h *Hub) LeaveRoom(roomID, sessionID string) {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if ok {
		room.mu.Lock()
		delete(room.sessions, sessionID)
		room.mu.Unlock()
	}
}

func (h *Hub) Broadcast(roomID string, senderID string, payload []byte) {
	h.mu.RLock()
	room, ok := h.rooms[roomID]
	h.mu.RUnlock()
	
	if !ok {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	for id, sess := range room.sessions {
		if id != senderID {
			select {
			case sess.Incoming <- payload:
			default:
			}
		}
	}
}
