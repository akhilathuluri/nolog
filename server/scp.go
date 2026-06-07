package server

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strings"
	"time"

	"secure-chat/manager"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/scp"
)

type HubFS struct {
	Hub *manager.Hub
}

func (h HubFS) Open(name string) (fs.File, error) {
	log.Printf("SCP Request for file: %q\n", name)
	data, ok := h.Hub.GetFile(name)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &memFile{
		name:   name,
		Reader: bytes.NewReader(data),
		size:   int64(len(data)),
		hub:    h.Hub,
	}, nil
}

func (h HubFS) Stat(name string) (fs.FileInfo, error) {
	data, ok := h.Hub.GetFile(name)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &memFile{
		name: name,
		size: int64(len(data)),
	}, nil
}

type memFile struct {
	name string
	*bytes.Reader
	size int64
	hub  *manager.Hub
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *memFile) Close() error {
	if f.hub != nil {
		f.hub.DeleteFile(f.name)
	}
	return nil
}

func (f *memFile) Name() string       { return f.name }
func (f *memFile) Size() int64        { return f.size }
func (f *memFile) Mode() fs.FileMode  { return 0444 }
func (f *memFile) ModTime() time.Time { return time.Now() }
func (f *memFile) IsDir() bool        { return false }
func (f *memFile) Sys() any           { return nil }

// HubWriteHandler implements scp.CopyFromClientHandler to receive files
type HubWriteHandler struct {
	Hub *manager.Hub
}

func (h HubWriteHandler) Mkdir(s ssh.Session, entry *scp.DirEntry) error {
	// Not supported/needed for ephemeral file sharing
	return fmt.Errorf("directories not supported")
}

func (h HubWriteHandler) Write(s ssh.Session, entry *scp.FileEntry) (int64, error) {
	info := scp.GetInfo(s.Command())
	// Expected format: upload_<UniqueID>
	if !strings.HasPrefix(info.Path, "upload_") {
		return 0, fmt.Errorf("invalid destination path: must be upload_<UniqueID>")
	}
	
	uid := strings.TrimPrefix(info.Path, "upload_")
	sess, ok := h.Hub.Get(uid)
	if !ok {
		return 0, fmt.Errorf("session not found")
	}

	if entry.Size > 10*1024*1024 {
		return 0, fmt.Errorf("file exceeds 10MB limit")
	}

	// Use LimitReader to physically prevent OOM if client lies about size
	data, err := io.ReadAll(io.LimitReader(entry.Reader, 10*1024*1024+1))
	if err != nil {
		return 0, err
	}
	if len(data) > 10*1024*1024 {
		return 0, fmt.Errorf("file physically exceeds 10MB limit")
	}

	payload := append([]byte("FILE:"+entry.Name+":"), data...)
	
	select {
	case sess.Uploads <- payload:
	default:
		return 0, fmt.Errorf("upload queue is full")
	}

	return int64(len(data)), nil
}
