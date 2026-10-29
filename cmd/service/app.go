package main

import (
	"fmt"
	"log/slog"

	"github.com/MishkaGql/template/cmd/service/server"
	"github.com/MishkaGql/template/config"
)

type App struct {
	server *server.Server
}

func NewApp(cfg *config.Config, server *server.Server) *App {
	return &App{
		server: server,
	}
}

func(app *App) Run() error {
	listener, err := app.server.Connect()
	if err != nil { 
		return fmt.Errorf("Listener setup failed: %w", err)
	}
		
	if err := app.server.Listen(listener); err != nil {
		slog.Error("Listen failed", "error", err)
	}

	if err = app.server.Shutdown(); err != nil {
		return fmt.Errorf("Shutdown failed: %w", err)
	}
	
	return nil
}

