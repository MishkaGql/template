package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-playground/validator/v10"

	"github.com/MishkaGql/template/internal/config"
	"github.com/MishkaGql/template/internal/container"
	"github.com/MishkaGql/template/internal/observability"
)

func main() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	cfg := config.MustLoad(validate)

	logger := observability.NewLogger(cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func run(cfg *config.Config, logger *slog.Logger) error {
	ctx := context.Background()

	otelProvider, err := observability.New(ctx, observability.ServiceName)
	if err != nil {
		return err
	}

	defer func() {
		if shutdownErr := otelProvider.Shutdown(context.WithoutCancel(ctx)); shutdownErr != nil {
			logger.Error("otel shutdown failed", "error", shutdownErr)
		}
	}()

	metrics, err := observability.NewMetrics(otelProvider.MeterProvider())
	if err != nil {
		return err
	}

	c, err := container.New(ctx, cfg, logger, metrics, otelProvider.MeterProvider())
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Pinger.Ping(ctx); err != nil {
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
		return err
	}

	notifyCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server started", "addr", server.Addr)

		if serveErr := server.Serve(listener); !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("listen failed", "error", serveErr)
			stop()
		}
	}()

	<-notifyCtx.Done()

	shutdownCtx, cancelTimeout := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancelTimeout()

	if err := server.Shutdown(shutdownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return closeErr
		}

		return err
	}

	return nil
}
