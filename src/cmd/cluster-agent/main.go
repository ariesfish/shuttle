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

	"inference-platform/internal/agent"
)

func main() {
	managementURL := flag.String("management-url", envOrDefault("AGENT_MANAGEMENT_URL", "http://localhost:8080"), "Management API base URL")
	clusterID := flag.String("cluster-id", os.Getenv("AGENT_CLUSTER_ID"), "Inference Cluster ID registered in the Management Plane")
	version := flag.String("version", envOrDefault("AGENT_VERSION", "dev"), "Cluster Agent version")
	authToken := flag.String("auth-token", os.Getenv("AGENT_AUTH_TOKEN"), "Bearer token for Management API auth")
	capabilities := flag.String("capability", os.Getenv("AGENT_CAPABILITIES"), "Comma-separated key=value capabilities")
	pollInterval := flag.Duration("poll-interval", durationEnvOrDefault("AGENT_POLL_INTERVAL", 5*time.Second), "Task polling interval")
	heartbeatInterval := flag.Duration("heartbeat-interval", durationEnvOrDefault("AGENT_HEARTBEAT_INTERVAL", 30*time.Second), "Heartbeat interval")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if strings.TrimSpace(*clusterID) == "" {
		logger.Error("cluster-id is required")
		os.Exit(2)
	}

	client := agent.NewManagementClient(*managementURL, &http.Client{Timeout: 30 * time.Second}).WithAuth(*authToken, "cluster-agent", "agent")
	runner := agent.NewRunner(client, agent.Config{
		ManagementURL:     *managementURL,
		ClusterID:         strings.TrimSpace(*clusterID),
		Version:           strings.TrimSpace(*version),
		Capabilities:      parseCapabilities(*capabilities),
		PollInterval:      *pollInterval,
		HeartbeatInterval: *heartbeatInterval,
	}, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("agent stopped", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func durationEnvOrDefault(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func parseCapabilities(value string) map[string]string {
	capabilities := map[string]string{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, capabilityValue, ok := strings.Cut(item, "=")
		if !ok {
			capabilities[item] = "true"
			continue
		}
		capabilities[strings.TrimSpace(key)] = strings.TrimSpace(capabilityValue)
	}
	if len(capabilities) == 0 {
		return nil
	}
	return capabilities
}
