package management

import (
	"testing"
	"time"
)

func TestAcceleratorPoolSummaryDerivesFromObservedInventory(t *testing.T) {
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
	pool, err := store.CreateAcceleratorPool(CreateAcceleratorPoolRequest{ClusterID: cluster.ID, Name: "h200-prod", NodeSelector: map[string]string{"pool": "h200"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Nodes: []AcceleratorInventoryNode{
			{Name: "node-a", Labels: map[string]string{"pool": "h200"}, Taints: []string{"nvidia.com/gpu:NoSchedule"}, Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: 8, MemoryMiB: 143360}}},
			{Name: "node-b", Labels: map[string]string{"pool": "h100"}, Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H100 SXM", DeviceCount: 8, MemoryMiB: 81920}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := store.ListAcceleratorPoolSummaries(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Pool.ID != pool.ID {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	summary := summaries[0]
	if summary.NodeCount != 1 || summary.AcceleratorCount != 8 || summary.AcceleratorModels["NVIDIA H200 SXM"] != 8 || summary.MemoryMiBSummary["NVIDIA H200 SXM"] != 143360 || summary.Freshness != AcceleratorInventoryFreshnessFresh || len(summary.Warnings) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestAcceleratorPoolSummaryWarnsForMissingEmptyAndStaleInventory(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateAcceleratorPool(CreateAcceleratorPoolRequest{ClusterID: cluster.ID, Name: "empty", NodeSelector: map[string]string{"pool": "missing"}})
	if err != nil {
		t.Fatal(err)
	}
	summaries, err := store.ListAcceleratorPoolSummaries(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Freshness != AcceleratorInventoryFreshnessMissing || summaries[0].Warnings[0] != "inventory missing" {
		t.Fatalf("unexpected missing summary: %+v", summaries)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now.Add(-10 * time.Minute) }
	_, err = store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: now.Add(-10 * time.Minute), Nodes: []AcceleratorInventoryNode{{Name: "node-a", Labels: map[string]string{"pool": "other"}}}})
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return now }
	summaries, err = store.ListAcceleratorPoolSummaries(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summaries[0].Freshness != AcceleratorInventoryFreshnessStale || summaries[0].NodeCount != 0 || len(summaries[0].Warnings) != 2 {
		t.Fatalf("unexpected stale empty summary: %+v", summaries[0])
	}
}

func TestInventoryDoesNotCreateAcceleratorPoolsOrAccessGrants(t *testing.T) {
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
	_, err = store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", Nodes: []AcceleratorInventoryNode{{Name: "node-a", Labels: map[string]string{"pool": "auto"}}}})
	if err != nil {
		t.Fatal(err)
	}
	pools, err := store.ListAcceleratorPools(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pools) != 0 {
		t.Fatalf("inventory must not create pools: %+v", pools)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	app, err := store.CreateServingApplication(CreateServingApplicationRequest{ProjectID: project.ID, Name: "app", Placement: PlacementIntent{ClusterID: cluster.ID, AcceleratorPoolID: "not-created", Namespace: "default"}, Model: ModelIntent{Family: "deepseek-v4", Variant: "flash", ArtifactID: "missing", Quantization: "fp8"}})
	if err == nil || app.ID != "" {
		t.Fatalf("expected existing serving app validation to remain responsible for app creation, app=%+v err=%v", app, err)
	}
}
