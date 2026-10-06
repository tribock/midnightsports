package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/tribock/midnightsports/api/internal/config"
	"github.com/tribock/midnightsports/api/internal/db"
	"github.com/tribock/midnightsports/api/internal/server"
)

// Werden beim Build gesetzt: -ldflags "-X main.version=... -X main.date=..."
var (
	version = "dev"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.Info("startup", "version", version, "date", date)

	// .env im Projektverzeichnis (make dev startet im Ordner api)
	cfg, err := config.Load("../.env", ".env")
	if err != nil {
		return err
	}
	if cfg.DevLogin {
		slog.Warn("DEV_LOGIN ist aktiv, Anmeldung ohne Google möglich")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	srv, err := server.New(cfg, pool, server.BuildInfo{Version: version, Date: date})
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdown)
	}()

	slog.Info("server started", "port", cfg.Port)
	if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
