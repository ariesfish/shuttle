package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zhiliu/internal/management"
)

func main() {
	addr := flag.String("addr", envOrDefault("MANAGEMENT_API_ADDR", ":8080"), "HTTP listen address")
	dataPath := flag.String("data", envOrDefault("MANAGEMENT_API_DATA", "data/management.json"), "JSON data file path")
	postgresDSN := flag.String("postgres-dsn", os.Getenv("MANAGEMENT_API_POSTGRES_DSN"), "Postgres DSN; when set, stores state in Postgres")
	authToken := flag.String("auth-token", os.Getenv("MANAGEMENT_API_AUTH_TOKEN"), "Bearer token for API auth; when empty, auth is disabled")
	corsOrigin := flag.String("cors-origin", envOrDefault("MANAGEMENT_API_CORS_ORIGIN", "*"), "Allowed CORS origin")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	var store management.ManagementStore
	var err error
	if *postgresDSN != "" {
		store, err = management.NewPostgresJSONStateStore(context.Background(), management.PostgresJSONStateOptions{DSN: *postgresDSN})
	} else {
		store, err = management.NewFileStore(*dataPath)
	}
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           management.NewServerWithOptions(store, logger, management.AuthConfig{Enabled: *authToken != "", Token: *authToken}, management.CORSConfig{AllowedOrigin: *corsOrigin}).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("management api listening", "addr", *addr, "data", *dataPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
