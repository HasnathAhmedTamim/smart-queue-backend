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

	"github.com/hasnathahmedtamim/smart-queue/internal/config"
	httpserver "github.com/hasnathahmedtamim/smart-queue/internal/http"
	"github.com/hasnathahmedtamim/smart-queue/internal/http/handlers"
	"github.com/hasnathahmedtamim/smart-queue/internal/realtime"
	"github.com/hasnathahmedtamim/smart-queue/internal/service"
	"github.com/hasnathahmedtamim/smart-queue/internal/storage/sqlite"
)

func main() {
	cfg := config.MustLoad()

	// DB
	db, err := sqlite.Open(cfg.Storage.Path)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Realtime hub
	hub := realtime.NewHub()

	// Service + handler
	queueSvc := service.NewQueueService(db.SQL, cfg.Queue.AvgServiceMinutes)
	queueHandler := handlers.NewQueueHandler(queueSvc, cfg.Queue.AdminKey, hub)

	// Router
	handler := httpserver.NewRouter(queueHandler, cfg.CORS.AllowedOrigin)

	// Server
	srv := &http.Server{
		Addr:    cfg.HTTP.Server.Address,
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("starting server", slog.String("addr", cfg.HTTP.Server.Address))

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", slog.String("error", err.Error()))
	}

	slog.Info("server stopped")
}
