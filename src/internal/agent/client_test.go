package agent

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"zhiliu/internal/management"
	platformtask "zhiliu/internal/task"
)

func previewTaskPayload(t *testing.T, clusterID string) map[string]any {
	t.Helper()
	envelope, err := platformtask.BuildRenderedDeploymentTask(platformtask.RenderedDeploymentTaskInput{
		Type:                 platformtask.TypePreviewDeploymentDiff,
		ServingApplicationID: "app-1",
		ClusterID:            clusterID,
		Resource:             platformtask.ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
		Manifests:            []platformtask.Manifest{{Name: "test.yaml", Content: "kind: ConfigMap\n"}},
	})
	if err != nil {
		t.Fatalf("build preview task payload: %v", err)
	}
	return platformtask.EncodePayload(envelope.Payload)
}

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

func TestClientRenewsTaskLease(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{ClusterID: cluster.ID, Type: management.TaskTypeInspectStatus})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	client := NewManagementClient(server.URL, server.Client())
	ctx := context.Background()
	agent, err := client.Register(ctx, management.RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	leased, ok, err := client.LeaseTask(ctx, cluster.ID, management.LeaseTaskRequest{AgentID: agent.ID})
	if err != nil || !ok {
		t.Fatalf("lease: task=%+v ok=%v err=%v", leased, ok, err)
	}
	renewed, err := client.RenewTaskLease(ctx, createdTask.ID, management.RenewTaskLeaseRequest{AgentID: agent.ID})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if renewed.ID != createdTask.ID || renewed.Status != management.TaskStatusLeased || !renewed.LeaseExpiresAt.After(leased.LeaseExpiresAt) {
		t.Fatalf("unexpected renewed task: %+v leased=%+v", renewed, leased)
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

func TestRunnerRenewsLeaseWhileTaskRuns(t *testing.T) {
	store, err := management.NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(management.CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(management.CreateTaskRequest{ClusterID: cluster.ID, Type: management.TaskTypeInspectStatus})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(management.NewServer(store, slog.Default()).Routes())
	defer server.Close()

	runner := NewRunnerWithExecutor(NewManagementClient(server.URL, server.Client()), Config{
		ClusterID:          cluster.ID,
		Version:            "test",
		PollInterval:       10 * time.Millisecond,
		HeartbeatInterval:  time.Second,
		LeaseRenewInterval: 20 * time.Millisecond,
	}, slog.Default(), slowExecutor{delay: 75 * time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = runner.Run(ctx)

	tasks, err := store.ListTasks(cluster.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ID != createdTask.ID || tasks[0].Status != management.TaskStatusSucceeded {
		t.Fatalf("expected slow task to complete, got %+v", tasks)
	}
	if tasks[0].Result["mode"] != "slow" {
		t.Fatalf("unexpected task result: %+v", tasks[0].Result)
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
		Payload:   previewTaskPayload(t, cluster.ID),
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

type slowExecutor struct {
	delay time.Duration
}

func (e slowExecutor) Execute(ctx context.Context, _ management.Task) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(e.delay):
		return map[string]any{"mode": "slow"}, nil
	}
}
