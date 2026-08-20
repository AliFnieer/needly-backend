package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AliFnieer/needly-backend/internal/cache"
	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/AliFnieer/needly-backend/internal/database"
	"github.com/AliFnieer/needly-backend/internal/observability"
	"github.com/AliFnieer/needly-backend/internal/server"
)

func main() {
	cfg := config.Load()

	// Initialize OpenTelemetry tracing
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	shutdownTracer, err := observability.InitTracerProvider(
		context.Background(),
		cfg.Tracing.ServiceName,
		otlpEndpoint,
	)
	if err != nil {
		slog.Error("failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(ctx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}()

	db, err := database.InitPostgres(cfg)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	if cfg.Server.GinMode == "release" {
		if err := database.RunMigrations(db, "migrations"); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	redisClient, err := cache.InitRedis(cfg)
	if err != nil {
		slog.Error("failed to initialize redis", "error", err)
		os.Exit(1)
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

	go func() {
		slog.Info("server starting", "addr", httpServer.Addr, "mode", cfg.Server.GinMode)

		tlsCert := os.Getenv("TLS_CERT_FILE")
		tlsKey := os.Getenv("TLS_KEY_FILE")

		if tlsCert != "" && tlsKey != "" {
			slog.Info("TLS enabled")
			if err := httpServer.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				slog.Error("TLS server failed", "error", err)
				os.Exit(1)
			}
		} else {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server failed to start", "error", err)
				os.Exit(1)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped gracefully")
}
