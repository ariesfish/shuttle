package management

import (
	"fmt"
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
	if inventory.Revision == "" || inventory.Freshness != AcceleratorInventoryFreshnessFresh || len(inventory.Nodes) != 1 || inventory.RevisionCount != 1 {
		t.Fatalf("unexpected inventory: %+v", inventory)
	}

	stored, err := store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}
	if stored.Revision != inventory.Revision || stored.Nodes[0].Accelerators[0].Product != "NVIDIA H200 SXM" || stored.RevisionCount != 1 {
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

func TestAcceleratorInventoryRevisionsAreIdempotentAndBounded(t *testing.T) {
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
	base := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		_, err := store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: base.Add(time.Duration(i) * time.Minute), Nodes: []AcceleratorInventoryNode{{Name: "node", Capacity: map[string]string{"nvidia.com/gpu": fmt.Sprintf("%d", i+1)}}}})
		if err != nil {
			t.Fatalf("report %d: %v", i, err)
		}
	}
	revisions, err := store.ListAcceleratorInventoryRevisions(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 10 {
		t.Fatalf("expected bounded 10 revisions, got %d", len(revisions))
	}
	latest := revisions[0]
	_, err = store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: latest.ObservedAt, Nodes: latest.Nodes, ProbeStatuses: latest.ProbeStatuses, CollectionMetadata: latest.CollectionMetadata})
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.ListAcceleratorInventoryRevisions(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(revisions) || after[0].Revision != latest.Revision {
		t.Fatalf("expected idempotent revision history, before=%+v after=%+v", revisions, after)
	}
}

func TestGetAcceleratorInventoryFreshnessStates(t *testing.T) {
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
		t.Fatalf("get unsupported inventory: %v", err)
	}
	if inventory.Freshness != AcceleratorInventoryFreshnessUnsupported || inventory.ClusterID != cluster.ID {
		t.Fatalf("unexpected unsupported inventory: %+v", inventory)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "phase-2"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.HeartbeatAgent(agent.ID, HeartbeatRequest{LastInventoryFreshness: string(AcceleratorInventoryFreshnessMissing)})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err = store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Freshness != AcceleratorInventoryFreshnessMissing {
		t.Fatalf("unexpected missing inventory: %+v", inventory)
	}
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now.Add(-10 * time.Minute) }
	_, err = store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: now.Add(-10 * time.Minute), Nodes: []AcceleratorInventoryNode{{Name: "node-a"}}})
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	inventory, err = store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Freshness != AcceleratorInventoryFreshnessStale {
		t.Fatalf("expected stale inventory, got %+v", inventory)
	}
}

func TestAcceleratorInventoryAuditAndRoutes(t *testing.T) {
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
	if fetched.Revision != reported.Revision || len(fetched.Nodes) != 1 || fetched.RevisionCount != 1 {
		t.Fatalf("unexpected fetched inventory: %+v", fetched)
	}
	revisions := requestJSON[[]AcceleratorInventory](t, server, http.MethodGet, "/v1/clusters/"+cluster.ID+"/accelerator-inventory/revisions", nil, http.StatusOK)
	if len(revisions) != 1 || revisions[0].Revision != reported.Revision {
		t.Fatalf("unexpected revisions: %+v", revisions)
	}
	records, err := store.ListAuditRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Action != "accelerator_inventory.report" || records[0].Metadata["revision"] != reported.Revision {
		t.Fatalf("unexpected audit records: %+v", records)
	}
}
