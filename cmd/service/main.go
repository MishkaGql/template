package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-playground/validator/v10"

	"github.com/MishkaGql/template/internal/config"
	"github.com/MishkaGql/template/internal/container"
)

func main() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	cfg := config.MustLoad(validate)

	if err := run(cfg); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

func run(cfg *config.Config) (err error) {
	ctx := context.Background()

	c, err := container.New(ctx, cfg)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           c.Router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		closeCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
		defer cancel()
		return errors.Join(fmt.Errorf("listen: %w", err), c.Close(closeCtx))
	}

	notifyCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c.Positions.Start(ctx)

	serveErrors := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", server.Addr)
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			serveErrors <- err
		}
	}()

	var serveErr error

	select {
	case <-notifyCtx.Done():
		slog.Info("server received shutdown signal")
	case err := <-serveErrors:
		serveErr = fmt.Errorf("serve http: %w", err)
	}

	shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelTimeout()

	httpErr := server.Shutdown(shutdownCtx)
	if httpErr != nil {
		slog.Warn("forcing HTTP shutdown", "error", httpErr)
		httpErr = errors.Join(httpErr, server.Close())
	}
	poolErr := c.Positions.Stop(shutdownCtx)
	return errors.Join(serveErr, httpErr, poolErr, c.Close(shutdownCtx))
}
