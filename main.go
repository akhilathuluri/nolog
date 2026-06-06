package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/scp"
	
	"secure-chat/manager"
	"secure-chat/server"
)

const (
	host = "localhost"
	port = "23234"
)

func main() {
	hub := manager.NewHub()

	s, err := wish.NewServer(
		wish.WithAddress(host+":"+port),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(server.TeaHandler(hub)),
			scp.Middleware(scp.NewFSReadHandler(server.HubFS{Hub: hub}), nil),
			logging.Middleware(),
		),
	)
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
