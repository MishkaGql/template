package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/MishkaGql/template/config"
)

type Server struct {
	server *http.Server
	cfg config.HTTPConfig
	shutdownTimeout time.Duration
}

func NewServer(
	cfg config.HTTPConfig,
	shutdownTimeout time.Duration, 
	router http.Handler,
) *Server{
	return &Server{
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           router,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},	
		cfg: cfg,
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Connect() (net.Listener, error) {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return nil, err
	}
	
	return listener, nil
}

func (s *Server) Listen(listener net.Listener) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	
	serveErr := make(chan error, 1)
	
	go func() {
		slog.Info("Server started...", "addr", s.server.Addr)
		if err := s.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()
	
	select {
		case <-ctx.Done():
			slog.Info("Server recived notify to stop server")
			return nil
		case err := <-serveErr:
			return fmt.Errorf("Server failed in serve: %w", err) 
	}
}

func(s *Server) Shutdown() error {
	ctx, cancelTimeout := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancelTimeout()
	
	if err := s.server.Shutdown(ctx); err != nil {
		slog.Info("Server shutdown timeout exceeded")

		if err := s.server.Close(); err != nil {
			return fmt.Errorf("Forced closing server failed: %w", err)
		}

		return err
	}
	
	return nil
}