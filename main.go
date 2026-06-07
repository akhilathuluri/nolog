package main

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/scp"
	"github.com/joho/godotenv"
	
	"secure-chat/manager"
	"secure-chat/server"
)

const (
	host = "localhost"
	port = "23234"
)

type encryptedLogger struct {
	file *os.File
	aead cipher.AEAD
	mu   sync.Mutex
}

func (e *encryptedLogger) Write(p []byte) (n int, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	nonce := make([]byte, e.aead.NonceSize())
	rand.Read(nonce)
	ciphertext := e.aead.Seal(nonce, nonce, p, nil)
	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(ciphertext)))
	e.file.Write(sizeBuf)
	e.file.Write(ciphertext)
	return len(p), nil
}

func decryptLogs(keyHex string) {
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil || len(keyBytes) != 32 {
		fmt.Println("Invalid log key. Must be a 64-character hex string.")
		os.Exit(1)
	}

	data, err := os.ReadFile("stealth.log")
	if err != nil {
		fmt.Printf("Could not read stealth.log: %v\n", err)
		os.Exit(1)
	}

	aead, err := chacha20poly1305.NewX(keyBytes)
	if err != nil {
		fmt.Printf("Failed to initialize cipher: %v\n", err)
		os.Exit(1)
	}

	nonceSize := aead.NonceSize()
	offset := 0
	
	fmt.Println("--- DECRYPTED TELEMETRY LOGS ---")
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		
		if offset+size > len(data) || size < nonceSize {
			break
		}
		
		ciphertext := data[offset : offset+size]
		offset += size
		
		nonce := ciphertext[:nonceSize]
		payload := ciphertext[nonceSize:]
		
		plaintext, err := aead.Open(nil, nonce, payload, nil)
		if err != nil {
			fmt.Printf("[DECRYPTION ERROR] %v\n", err)
			continue
		}
		
		fmt.Print(string(plaintext))
	}
	fmt.Println("--- END OF LOGS ---")
}

func main() {
	var cliPass string
	var readLogsKey string
	flag.StringVar(&cliPass, "password", "", "Server password for SSH authentication")
	flag.StringVar(&readLogsKey, "read-logs", "", "Decrypt and print stealth.log using the provided hex key")
	flag.Parse()

	if readLogsKey != "" {
		decryptLogs(readLogsKey)
		return
	}

	logKey := make([]byte, 32)
	rand.Read(logKey)
	fmt.Printf("[SYS] 🛡️  Stealth Engine Logger Initialized\n")
	fmt.Printf("[SYS] 🔑 Session Log Key: %x\n", logKey)
	fmt.Printf("[SYS] All telemetry is securely encrypted in 'stealth.log'\n")

	logFile, _ := os.OpenFile("stealth.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	aead, _ := chacha20poly1305.NewX(logKey)

	log.SetOutput(&encryptedLogger{
		file: logFile,
		aead: aead,
	})

	hub := manager.NewHub()

	opts := []ssh.Option{
		wish.WithAddress(host + ":" + port),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(server.TeaHandler(hub)),
			scp.Middleware(scp.NewFSReadHandler(server.HubFS{Hub: hub}), server.HubWriteHandler{Hub: hub}),
			logging.Middleware(),
		),
	}

	_ = godotenv.Load() // Load .env file if it exists, ignore if not

	expectedPass := cliPass
	if expectedPass == "" {
		expectedPass = os.Getenv("SECURE_CHAT_PASS")
	}

	if expectedPass != "" {
		opts = append(opts, wish.WithPasswordAuth(func(ctx ssh.Context, password string) bool {
			return password == expectedPass
		}))
	}

	s, err := wish.NewServer(opts...)
	if err != nil {
		log.Fatalln("Could not start server", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%s", host, port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatalln("Could not start server", err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Fatalln("Could not stop server", err)
	}
}
