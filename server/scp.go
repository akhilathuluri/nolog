package server

import (
	"bytes"
	"io/fs"
	"log"
	"time"

	"secure-chat/manager"
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
