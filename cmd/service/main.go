package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	spretty "github.com/mickamy/slog-pretty"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/cmd/service/server"
	"github.com/MishkaGql/template/config"
	"github.com/MishkaGql/template/internal"
	"github.com/MishkaGql/template/internal/db"
	"github.com/MishkaGql/template/internal/errs"

	"github.com/MishkaGql/template/internal/handler"
	"github.com/MishkaGql/template/internal/repository"
)


func main() {
	// сейчас валидирует только конфиг 
	validate := validator.New(validator.WithRequiredStructEnabled())
	cfg := config.MustLoad(validate)
	
	logger := setupLogger(cfg.LogLevel)
	slog.SetDefault(logger)	
	
	slog.Info("Configuration loaded", "config", cfg)

	if err := run(cfg); err != nil {
		slog.Error("Application shutdown failed", "error", err)
		os.Exit(1)
	}
	
	slog.Info("Application gracefully stopped") 
	
}

func run(cfg *config.Config) error {
	// БАЗА ДАННЫХ
    pool, err := db.NewDbPool(context.Background(), cfg.DB)
    if err != nil {
    	return fmt.Errorf("Pool init failed: %w",err)
    }
	defer pool.Close()

	// Сразу падаем при неудачном пинге
	pinger := db.NewDbPinger(pool, cfg.DB.PingTimeout)
	if err = pinger.Ping(context.Background()); err != nil {
    	return fmt.Errorf("Ping db on startup failed: %w", err)
    }
    
    // ЗАВИСИМОСТИ
	repo := repository.NewRepository(pool, cfg.DB.QueryTimeout)
    tx := db.NewPgxTxManager(pool)
    
    svc := internal.NewService(repo,tx)
    
    handler := handler.NewHandler(svc, pinger)
	router := setupRouter(handler)

	
 	// ИНИЦИАЛИЗАЦИЯ СЕРВЕРА
	server := server.NewServer(cfg.HTTP, cfg.ShutdownTimeout, router)
	app := NewApp(cfg, server)

	if err := app.Run(); err != nil {
		return err
	}
	
	return nil
}

func setupRouter(h *handler.TripHandler) http.Handler {
	loader := openapi3.NewLoader()
	
	// хардкод - пойдет пока что
	spec, _ := loader.LoadFromFile("contracts/openapi/trip-service.openapi.yaml")
	spec.Servers = nil

	// валидирует всё по спеке
	specValidator := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		ErrorHandler: func(w http.ResponseWriter, message string, statusCode int) {
			handler.SendError(w,errs.ValidationServiceError)
		},
	})
	
	router := chi.NewRouter()
	router.Use(specValidator)
	
	handler := api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter: router,
		
		// Для валидации params в path.
		// specValidator не работает там
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handler.SendError(w,errs.ValidationServiceError)
		},
	})
	
	return handler
}

func setupLogger(envLvl string) *slog.Logger{
	var lvl slog.Level
	
	if err := lvl.UnmarshalText([]byte(envLvl)); err != nil {
		panic(fmt.Sprintf("log level parsing error: %v", err))
	}

	handler := spretty.NewHandler(
		os.Stdout,
		&spretty.HandlerOptions{
			Level:     lvl,
		},
	)
	
	return slog.New(handler)
}

