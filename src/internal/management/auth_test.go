package management

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthRejectsMissingBearerToken(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServerWithAuth(store, slog.Default(), AuthConfig{Enabled: true, Token: "secret"}).Routes())
	defer server.Close()

	resp, err := server.Client().Post(server.URL+"/v1/projects", "application/json", bytes.NewBufferString(`{"name":"platform"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthAllowsAdminAndRecordsAudit(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewServerWithAuth(store, slog.Default(), AuthConfig{Enabled: true, Token: "secret"}).Routes())
	defer server.Close()

	body := bytes.NewBufferString(`{"name":"platform"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/projects", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Actor", "alice")
	req.Header.Set("X-Role", "admin")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	records, err := store.ListAuditRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Actor != "alice" || records[0].Action != "create_project" {
		encoded, _ := json.Marshal(records)
		t.Fatalf("unexpected audit records: %s", encoded)
	}
}
