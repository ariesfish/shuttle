package management

import (
	"errors"
	"testing"
	"time"
)

func TestProjectClusterAgentAndTaskLifecycle(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}

	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID == "" || project.Name != "platform" {
		t.Fatalf("unexpected project: %+v", project)
	}

	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	agent, err := store.RegisterAgent(RegisterAgentRequest{
		ClusterID: cluster.ID,
		Version:   "v0.1.0",
		Capabilities: map[string]string{
			"dynamo": "true",
		},
	})
	if err != nil {
		t.Fatalf("register agent: %v", err)
	}
	if agent.ClusterID != cluster.ID || agent.LastHeartbeat.IsZero() {
		t.Fatalf("unexpected agent: %+v", agent)
	}

	task, err := store.CreateTask(CreateTaskRequest{
		ClusterID: cluster.ID,
		Type:      TaskTypeInspectStatus,
		Payload: map[string]any{
			"servingApplicationId": "sa-1",
		},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("new task status = %q", task.Status)
	}

	leased, err := store.LeaseNextTask(cluster.ID, LeaseTaskRequest{AgentID: agent.ID}, time.Minute)
	if err != nil {
		t.Fatalf("lease task: %v", err)
	}
	if leased.ID != task.ID || leased.Status != TaskStatusLeased || leased.LeaseOwner != agent.ID {
		t.Fatalf("unexpected lease: %+v", leased)
	}

	completed, err := store.CompleteTask(task.ID, CompleteTaskRequest{
		AgentID: agent.ID,
		Status:  TaskStatusSucceeded,
		Result: map[string]any{
			"phase": "Ready",
		},
	})
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if completed.Status != TaskStatusSucceeded || completed.Result["phase"] != "Ready" {
		t.Fatalf("unexpected completed task: %+v", completed)
	}
}

func TestRenewTaskLeaseExtendsOwnedLease(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(CreateTaskRequest{ClusterID: cluster.ID, Type: TaskTypeInspectStatus})
	if err != nil {
		t.Fatal(err)
	}
	leased, err := store.LeaseNextTask(cluster.ID, LeaseTaskRequest{AgentID: agent.ID}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return leased.LeaseExpiresAt.Add(-500 * time.Millisecond) }
	renewed, err := store.RenewTaskLease(createdTask.ID, RenewTaskLeaseRequest{AgentID: agent.ID}, time.Minute)
	if err != nil {
		t.Fatalf("renew task lease: %v", err)
	}
	if !renewed.LeaseExpiresAt.After(leased.LeaseExpiresAt) || renewed.LeaseOwner != agent.ID || renewed.Status != TaskStatusLeased {
		t.Fatalf("unexpected renewed task: %+v original=%+v", renewed, leased)
	}
}

func TestCompleteTaskIsIdempotentAfterTerminalState(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID})
	if err != nil {
		t.Fatal(err)
	}
	createdTask, err := store.CreateTask(CreateTaskRequest{ClusterID: cluster.ID, Type: TaskTypeInspectStatus})
	if err != nil {
		t.Fatal(err)
	}
	leased, err := store.LeaseNextTask(cluster.ID, LeaseTaskRequest{AgentID: agent.ID}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.CompleteTask(createdTask.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusSucceeded, Result: map[string]any{"attempt": "first"}})
	if err != nil {
		t.Fatalf("first complete: %v", err)
	}
	second, err := store.CompleteTask(createdTask.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusSucceeded, Result: map[string]any{"attempt": "second"}})
	if err != nil {
		t.Fatalf("second complete: %v", err)
	}
	if second.Result["attempt"] != "first" || !second.UpdatedAt.Equal(first.UpdatedAt) || leased.ID != second.ID {
		t.Fatalf("expected idempotent complete to preserve terminal task, first=%+v second=%+v", first, second)
	}
	_, err = store.CompleteTask(createdTask.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusFailed})
	if !errors.Is(err, ErrTaskLeaseHeld) {
		t.Fatalf("expected conflicting terminal complete to fail, got %v", err)
	}
}

func TestLeaseRejectsAgentFromAnotherCluster(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}

	clusterA, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a"})
	if err != nil {
		t.Fatal(err)
	}
	clusterB, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-b"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: clusterB.ID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateTask(CreateTaskRequest{ClusterID: clusterA.ID, Type: TaskTypeInspectStatus})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.LeaseNextTask(clusterA.ID, LeaseTaskRequest{AgentID: agent.ID}, time.Minute)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateServingApplicationAndPreviewTask(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a", PrometheusURL: "http://prometheus.local", GrafanaURL: "http://grafana.local"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.CreateModelArtifact(CreateModelArtifactRequest{
		Family:        "deepseek-v4",
		Variant:       "flash",
		Revision:      "rev1",
		PVCMountPath:  "/models",
		PVCModelPath:  "hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1",
		HostCachePath: "/data/models/hub",
		Quantization:  "fp8",
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := store.CreateServingApplication(CreateServingApplicationRequest{
		ProjectID: project.ID,
		Name:      "DeepSeek V4 Flash",
		Model: ModelIntent{
			Family:       "deepseek-v4",
			Variant:      "flash",
			ArtifactID:   artifact.ID,
			Quantization: "fp8",
		},
		Placement: PlacementIntent{ClusterID: cluster.ID, Namespace: "tenant-a"},
		Runtime: RuntimeIntent{
			Backend:  "vllm",
			Topology: "pd-disagg",
			Recipe:   "deepseek-v4-flash-vllm-dgd-disagg",
		},
		Service:      ServiceIntent{EndpointName: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
		Optimization: OptimizationIntent{Target: "throughput", ProfilingMode: "disabled"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.Phase != ServingApplicationPhaseDraft {
		t.Fatalf("unexpected app phase: %+v", app)
	}
	observability, err := store.GetObservabilityEntry(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if observability.GrafanaURL == "" || observability.PrometheusURL == "" || len(observability.PrometheusQueries) == 0 {
		t.Fatalf("unexpected observability entry: %+v", observability)
	}

	task, err := store.CreatePreviewTask(CreatePreviewTaskRequest{ServingApplicationID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if task.Type != TaskTypePreviewDeploymentDiff || task.ClusterID != cluster.ID {
		t.Fatalf("unexpected preview task: %+v", task)
	}
	manifests, ok := task.Payload["manifests"].([]any)
	if !ok || len(manifests) != 1 {
		t.Fatalf("unexpected manifests payload: %+v", task.Payload)
	}

	applyTask, err := store.CreateApplyTask(CreateApplyTaskRequest{ServingApplicationID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if applyTask.Type != TaskTypeApplyDeployment || applyTask.Payload["resourceName"] != "deepseek-v4-flash" || applyTask.Payload["namespace"] != "tenant-a" {
		t.Fatalf("unexpected apply task: %+v", applyTask)
	}
	transitions, err := store.ListServingApplicationTransitions(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(transitions) != 2 || transitions[0].To != ServingApplicationPhaseDraft || transitions[1].To != ServingApplicationPhaseApplying || transitions[1].TaskID != applyTask.ID {
		t.Fatalf("unexpected transitions after apply task: %+v", transitions)
	}
	updatedApp, err := store.GetServingApplication(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedApp.Phase != ServingApplicationPhaseApplying {
		t.Fatalf("expected app phase Applying, got %+v", updatedApp)
	}

	redeployTask, err := store.CreateRedeployTask(CreateRedeployTaskRequest{ServingApplicationID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if redeployTask.Type != TaskTypeDeleteBeforeApply || redeployTask.Payload["resourceName"] != "deepseek-v4-flash" {
		t.Fatalf("unexpected redeploy task: %+v", redeployTask)
	}

	retireTask, err := store.CreateRetireTask(CreateRetireTaskRequest{ServingApplicationID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if retireTask.Type != TaskTypeRetireDeployment || retireTask.Payload["namespace"] != "tenant-a" {
		t.Fatalf("unexpected retire task: %+v", retireTask)
	}
	diagnosticsTask, err := store.CreateDiagnosticsTask(CreateDiagnosticsTaskRequest{ServingApplicationID: app.ID})
	if err != nil {
		t.Fatal(err)
	}
	if diagnosticsTask.Type != TaskTypeFetchDiagnostics || diagnosticsTask.Payload["resourceName"] != "deepseek-v4-flash" || diagnosticsTask.Payload["namespace"] != "tenant-a" {
		t.Fatalf("unexpected diagnostics task: %+v", diagnosticsTask)
	}
	retiringApp, err := store.GetServingApplication(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retiringApp.Phase != ServingApplicationPhaseRetiring {
		t.Fatalf("expected app phase Retiring, got %+v", retiringApp)
	}

	_, err = store.CompleteTask(task.ID, CompleteTaskRequest{AgentID: "agent-1", Status: TaskStatusSucceeded})
	if !errors.Is(err, ErrTaskLeaseHeld) {
		t.Fatalf("expected lease owner check, got %v", err)
	}

	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID})
	if err != nil {
		t.Fatal(err)
	}
	leasedPreview, err := forceLeaseTask(store, cluster.ID, task.ID, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CompleteTask(leasedPreview.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusSucceeded})
	if err != nil {
		t.Fatal(err)
	}
	validatedApp, err := store.GetServingApplication(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if validatedApp.Phase != ServingApplicationPhaseValidated {
		t.Fatalf("expected app phase Validated, got %+v", validatedApp)
	}

	leasedApply, err := forceLeaseTask(store, cluster.ID, applyTask.ID, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CompleteTask(leasedApply.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusSucceeded, Result: map[string]any{"phase": "Ready"}})
	if err != nil {
		t.Fatal(err)
	}
	readyApp, err := store.GetServingApplication(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readyApp.Phase != ServingApplicationPhaseReady || readyApp.EndpointURL == "" {
		t.Fatalf("expected app Ready with endpoint, got %+v", readyApp)
	}
	transitions, err = store.ListServingApplicationTransitions(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	lastTransition := transitions[len(transitions)-1]
	if lastTransition.To != ServingApplicationPhaseReady || lastTransition.Actor != agent.ID || lastTransition.TaskID != leasedApply.ID {
		t.Fatalf("unexpected ready transition: %+v", lastTransition)
	}
	endpoints, err := store.ListEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].ServingApplicationID != app.ID || !endpoints[0].Ready {
		t.Fatalf("unexpected endpoints: %+v", endpoints)
	}

	leasedRetire, err := forceLeaseTask(store, cluster.ID, retireTask.ID, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CompleteTask(leasedRetire.ID, CompleteTaskRequest{AgentID: agent.ID, Status: TaskStatusSucceeded})
	if err != nil {
		t.Fatal(err)
	}
	retiredApp, err := store.GetServingApplication(app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retiredApp.Phase != ServingApplicationPhaseRetired {
		t.Fatalf("expected app phase Retired, got %+v", retiredApp)
	}
	endpoints, err = store.ListEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 0 {
		t.Fatalf("expected endpoint cleanup, got %+v", endpoints)
	}
}

func forceLeaseTask(store *FileStore, clusterID, taskID, agentID string) (Task, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	task := store.data.Tasks[taskID]
	task.Status = TaskStatusLeased
	task.LeaseOwner = agentID
	task.LeaseExpiresAt = store.now().UTC().Add(time.Minute)
	task.UpdatedAt = store.now().UTC()
	store.data.Tasks[taskID] = task
	return task, nil
}

func TestTaskTypeWhitelist(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CreateTask(CreateTaskRequest{ClusterID: cluster.ID, Type: TaskType("ArbitraryKubectl")})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
