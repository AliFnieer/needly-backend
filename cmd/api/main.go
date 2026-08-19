package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/database"
	"github.com/AliFnieer/needly-backend/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.InitPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	redisClient, err := cache.InitRedis(cfg)
	if err != nil {
		log.Fatalf("failed to initialize redis: %v", err)
	}
	defer redisClient.Close()

	srv := server.NewServer(cfg, db, redisClient)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      srv.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("server starting", "addr", httpServer.Addr, "mode", cfg.Server.GinMode)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down", "signal", sig.String())

	// Give in-flight requests up to 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
