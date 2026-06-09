package agent

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"zhiliu/internal/management"
)

func TestClientReportsAcceleratorInventory(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	agentRecord, err := store.RegisterAgent(management.RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	client := NewManagementClient(server.URL, server.Client())
	inventory, err := client.ReportAcceleratorInventory(context.Background(), cluster.ID, management.ReportAcceleratorInventoryRequest{
		AgentID:       agentRecord.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Nodes:         []management.AcceleratorInventoryNode{{Name: "fake-node-1"}},
	})
	if err != nil {
		t.Fatalf("report inventory: %v", err)
	}
	if inventory.Revision == "" || inventory.Freshness != management.AcceleratorInventoryFreshnessFresh {
		t.Fatalf("unexpected inventory: %+v", inventory)
	}
}

func TestRunnerReportsFakeInventoryAndCompletesPhase1Tasks(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{ClusterID: cluster.ID, Type: "InspectDeploymentStatus"})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	runner := NewRunnerWithInventory(NewManagementClient(server.URL, server.Client()), Config{
		ClusterID:         cluster.ID,
		Version:           "test",
		PollInterval:      10 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond,
	}, slog.Default(), FakeKubernetesExecutor{}, FakeInventoryReporter{Now: func() time.Time {
		return time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = runner.Run(ctx)

	inventory, err := store.GetAcceleratorInventory(cluster.ID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}
	if inventory.Freshness != management.AcceleratorInventoryFreshnessFresh || len(inventory.Nodes) != 1 || inventory.Nodes[0].Name != "fake-node-1" {
		t.Fatalf("unexpected fake inventory: %+v", inventory)
	}
	if len(inventory.Nodes[0].Connectivity) != 2 || inventory.Nodes[0].Connectivity[1].Type != "rdma" || !inventory.Nodes[0].Connectivity[1].Present {
		t.Fatalf("expected fake inventory to include RDMA connectivity: %+v", inventory.Nodes[0].Connectivity)
	}
	tasks, err := store.ListTasks(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != createdTask.ID || tasks[0].Status != management.TaskStatusSucceeded {
		t.Fatalf("phase 1 task compatibility failed: %+v", tasks)
	}
	agents, err := store.ListAgents()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].LastInventoryRevision == "" || agents[0].LastInventoryFreshness != string(management.AcceleratorInventoryFreshnessFresh) {
		t.Fatalf("agent inventory heartbeat linkage missing: %+v", agents)
	}
}
