package management

import (
	"net/http"
	"testing"
	"time"
)

func TestProductionObservabilityEntryPointsExposeLinksAndWarnings(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a", PrometheusURL: "http://prometheus.example", GrafanaURL: "http://grafana.example"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC), Nodes: []AcceleratorInventoryNode{{Name: "node-a"}}, ProbeStatuses: []AcceleratorInventoryProbe{{Name: "nvidia-dcgm", Status: "ok"}}})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.GetProductionObservabilityEntryPoints(cluster.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.InventoryRevision != inventory.Revision || len(entry.Links) < 3 || len(entry.Alerts) != 0 || len(entry.TelemetryCoverage) != 0 {
		t.Fatalf("unexpected entry points: %+v", entry)
	}
}

func TestProductionObservabilityEntryPointsWarnForMissingTelemetry(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := store.GetProductionObservabilityEntryPoints(cluster.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.Alerts) != 3 || len(entry.TelemetryCoverage) != 3 {
		t.Fatalf("expected missing grafana/prometheus/inventory warnings, got %+v", entry)
	}
}

func TestProductionObservabilityRoutes(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil).Routes()
	entry := requestJSON[ProductionObservabilityEntryPoints](t, server, http.MethodGet, "/v1/clusters/"+cluster.ID+"/observability/entry-points", nil, http.StatusOK)
	if entry.ClusterID != cluster.ID || len(entry.Alerts) == 0 {
		t.Fatalf("unexpected route entry: %+v", entry)
	}
}
