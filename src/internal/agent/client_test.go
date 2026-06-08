package agent

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"inference-platform/internal/management"
)

func TestClientRegistersLeasesAndCompletesNoopTask(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{
		ClusterID: cluster.ID,
		Type:      management.TaskTypeInspectStatus,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	client := NewManagementClient(server.URL, server.Client())
	ctx := context.Background()
	agent, err := client.Register(ctx, management.RegisterAgentRequest{
		ClusterID: cluster.ID,
		Version:   "test",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	leased, ok, err := client.LeaseTask(ctx, cluster.ID, management.LeaseTaskRequest{AgentID: agent.ID})
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if !ok {
		t.Fatal("expected task to be leased")
	}
	if leased.ID != createdTask.ID || leased.LeaseOwner != agent.ID {
		t.Fatalf("unexpected leased task: %+v", leased)
	}

	completed, err := client.CompleteTask(ctx, leased.ID, management.CompleteTaskRequest{
		AgentID: agent.ID,
		Status:  management.TaskStatusSucceeded,
		Result: map[string]any{
			"mode": "noop",
		},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Status != management.TaskStatusSucceeded {
		t.Fatalf("unexpected completed task: %+v", completed)
	}
}

func TestRunnerCompletesNoopTask(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{
		ClusterID: cluster.ID,
		Type:      management.TaskTypeInspectStatus,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	runner := NewRunner(NewManagementClient(server.URL, server.Client()), Config{
		ClusterID:         cluster.ID,
		Version:           "test",
		PollInterval:      10 * time.Millisecond,
		HeartbeatInterval: time.Second,
	}, slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = runner.Run(ctx)

	tasks, err := store.ListTasks(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != createdTask.ID {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	if tasks[0].Status != management.TaskStatusSucceeded {
		t.Fatalf("expected runner to complete task, got %+v", tasks[0])
	}
}

func TestRunnerCompletesPreviewDeploymentDiffTask(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{
		ClusterID: cluster.ID,
		Type:      management.TaskTypePreviewDeploymentDiff,
		Payload: map[string]any{
			"manifests": []any{map[string]any{"name": "test.yaml", "content": "kind: ConfigMap\n"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	runner := NewRunnerWithExecutor(NewManagementClient(server.URL, server.Client()), Config{
		ClusterID:         cluster.ID,
		Version:           "test",
		PollInterval:      10 * time.Millisecond,
		HeartbeatInterval: time.Second,
	}, slog.Default(), NewTaskExecutor(&fakeDryRunner{result: DryRunResult{Stdout: "dry-run ok"}}))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = runner.Run(ctx)

	tasks, err := store.ListTasks(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != createdTask.ID {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	if tasks[0].Status != management.TaskStatusSucceeded {
		t.Fatalf("expected runner to complete task, got %+v", tasks[0])
	}
	if tasks[0].Result["mode"] != "server-side-dry-run" || tasks[0].Result["stdout"] != "dry-run ok" {
		t.Fatalf("unexpected result: %+v", tasks[0].Result)
	}
}
