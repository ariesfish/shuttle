package management

import (
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestReportAndGetAcceleratorInventory(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)

	inventory, err := store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    observedAt,
		Nodes: []AcceleratorInventoryNode{
			{
				Name:        "fake-node-1",
				Labels:      map[string]string{"nvidia.com/gpu.product": "NVIDIA-H200-SXM"},
				Capacity:    map[string]string{"nvidia.com/gpu": "8"},
				Allocatable: map[string]string{"nvidia.com/gpu": "8"},
				Accelerators: []AcceleratorInventoryAccelerator{
					{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: 8, MemoryMiB: 143360},
				},
			},
		},
		ProbeStatuses: []AcceleratorInventoryProbe{{Name: "fake-inventory", Status: "ok"}},
	})
	if err != nil {
		t.Fatalf("report inventory: %v", err)
	}
	if inventory.Revision == "" || inventory.Freshness != AcceleratorInventoryFreshnessFresh || len(inventory.Nodes) != 1 {
		t.Fatalf("unexpected inventory: %+v", inventory)
	}

	stored, err := store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}
	if stored.Revision != inventory.Revision || stored.Nodes[0].Accelerators[0].Product != "NVIDIA H200 SXM" {
		t.Fatalf("unexpected stored inventory: %+v", stored)
	}
	agents, err := store.ListAgents()
	if err != nil {
		t.Fatal(err)
	}
	if agents[0].LastInventoryRevision != inventory.Revision || agents[0].LastInventoryFreshness != string(AcceleratorInventoryFreshnessFresh) || !agents[0].LastInventoryObservedAt.Equal(observedAt) {
		t.Fatalf("agent inventory linkage missing: %+v", agents[0])
	}
}

func TestGetAcceleratorInventoryMissingIsCompatible(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "phase-1-cluster"})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatalf("get missing inventory: %v", err)
	}
	if inventory.Freshness != AcceleratorInventoryFreshnessMissing || inventory.ClusterID != cluster.ID {
		t.Fatalf("unexpected missing inventory: %+v", inventory)
	}
}

func TestAcceleratorInventoryRoutes(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, slog.Default()).Routes()

	reported := requestJSON[AcceleratorInventory](t, server, http.MethodPost, "/v1/clusters/"+cluster.ID+"/accelerator-inventory", ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Nodes:         []AcceleratorInventoryNode{{Name: "fake-node-1"}},
	}, http.StatusOK)
	if reported.Revision == "" || reported.Freshness != AcceleratorInventoryFreshnessFresh {
		t.Fatalf("unexpected reported inventory: %+v", reported)
	}

	fetched := requestJSON[AcceleratorInventory](t, server, http.MethodGet, "/v1/clusters/"+cluster.ID+"/accelerator-inventory", nil, http.StatusOK)
	if fetched.Revision != reported.Revision || len(fetched.Nodes) != 1 {
		t.Fatalf("unexpected fetched inventory: %+v", fetched)
	}
}
