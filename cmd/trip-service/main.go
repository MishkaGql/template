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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/config"
	"github.com/MishkaGql/template/internal"
	"github.com/MishkaGql/template/internal/db"
	"github.com/MishkaGql/template/internal/errs"
	"github.com/MishkaGql/template/internal/handler"
	"github.com/MishkaGql/template/internal/repository"
)

var validate *validator.Validate

func main() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	config := config.MustLoad(validate)

	logger := setupLogger(config.LogLevel)
	slog.SetDefault(logger)

	if err := run(config); err != nil {
		slog.Error("Server unsuccessfully closed", "error", err)
		os.Exit(1)
	}

	slog.Info("Server successfully closed")
}

func run(cfg *config.Config) error {
    pool, err := db.NewDbPool(context.Background(), cfg.DB)
    if err != nil {
    	slog.Error("Pool init failed")
    	return err
    }
	defer pool.Close()

	pinger := db.NewDbPinger(pool, cfg.DB.PingTimeout)

    err = pinger.Ping(context.Background())
    if err != nil {
        slog.Error("Ping db on startup failed")
    	return err
    }

	r := repository.NewRepository(pool, cfg.DB.QueryTimeout)
    tx := db.NewPgxTxManager(pool)
    s := internal.NewService(r,tx)
    h := handler.NewHandler(s, pinger)
	handler := setupHandler(h)

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		slog.Error("Listener setup failed")
		return err
	}

	// Gracefull Shutdown
	notifyCtx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		slog.Info("Server started...", "addr", server.Addr)
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Listening Server error")
			cancel()
		}
	}()

	<-notifyCtx.Done()

	shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelTimeout()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Info("Server shutdown timeout exceeded")

		if err := server.Close(); err != nil {
			slog.Error("Forced closing server failed")
			return err
		}

		return err
	}

	return nil
}

func setupHandler(h *handler.TripHandler) http.Handler {
	loader := openapi3.NewLoader()

	spec, _ := loader.LoadFromFile("contracts/openapi/trip-service.openapi.yaml")
	spec.Servers = nil

	// валидирует всё по спеке
	mw := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			handler.SendError(w, errs.ValidationServiceError)
		},
	})

	router := chi.NewRouter()
	router.Use(mw)

	return api.HandlerFromMux(h, router)
}

func setupLogger(envLvl string) *slog.Logger {
	var lvl slog.Level
	lvl.UnmarshalText([]byte(envLvl))

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: lvl},
	)

	return slog.New(handler)
}

