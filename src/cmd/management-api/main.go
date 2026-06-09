package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"zhiliu/internal/management"
)

func main() {
	addr := flag.String("addr", envOrDefault("MANAGEMENT_API_ADDR", ":8080"), "HTTP listen address")
	dataPath := flag.String("data", envOrDefault("MANAGEMENT_API_DATA", "data/management.json"), "JSON data file path")
	postgresDSN := flag.String("postgres-dsn", os.Getenv("MANAGEMENT_API_POSTGRES_DSN"), "Postgres DSN; when set, stores state in Postgres")
	authToken := flag.String("auth-token", os.Getenv("MANAGEMENT_API_AUTH_TOKEN"), "Bearer token for API auth; when empty and auth-tokens is empty, auth is disabled")
	authTokens := flag.String("auth-tokens", os.Getenv("MANAGEMENT_API_TOKENS"), "Comma-separated token:actor:role entries for API auth")
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

	authConfig := management.AuthConfig{Enabled: *authToken != "" || *authTokens != "", Token: *authToken, Tokens: parseAuthTokens(*authTokens)}
	server := &http.Server{
		Addr:              *addr,
		Handler:           management.NewServerWithOptions(store, logger, authConfig, management.CORSConfig{AllowedOrigin: *corsOrigin}).Routes(),
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

func parseAuthTokens(value string) map[string]management.Actor {
	actors := map[string]management.Actor{}
	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		token := strings.TrimSpace(parts[0])
		if token == "" {
			continue
		}
		actor := management.Actor{Name: "api-token", Role: "viewer"}
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			actor.Name = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
			actor.Role = strings.TrimSpace(parts[2])
		}
		actors[token] = actor
	}
	return actors
}
