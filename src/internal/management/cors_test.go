package management

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSPreflightBypassesAuth(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServerWithOptions(store, slog.Default(), AuthConfig{Enabled: true, Token: "secret"}, CORSConfig{AllowedOrigin: "*"}).Routes())
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/v1/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type,authorization,x-actor,x-role")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin: %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}
