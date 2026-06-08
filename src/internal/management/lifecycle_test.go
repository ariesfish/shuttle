package management

import (
	"context"
	"testing"
	"time"

	platformtask "zhiliu/internal/task"
)

func TestServingApplicationLifecycleRequestsApplyAction(t *testing.T) {
	repo := &fakeLifecycleRepository{actionState: lifecycleTestActionState()}
	lifecycle := NewServingApplicationLifecycleWithOptions(repo, fakeDeploymentRenderer{}, platformtask.DefaultRegistry())

	task, err := lifecycle.RequestAction(context.Background(), "app-1", ServingApplicationActionApply, "operator-1")
	if err != nil {
		t.Fatalf("request action: %v", err)
	}
	if task.Type != platformtask.TaskTypeApplyDeployment || task.ClusterID != "cluster-1" || task.Payload["resourceName"] != "deepseek-v4-flash" || task.Payload["namespace"] != "dynamo-system" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if repo.requested.TransitionPhase != ServingApplicationPhaseApplying || repo.requested.TransitionReason != "apply task created" || repo.requested.Actor != "operator-1" {
		t.Fatalf("unexpected requested action: %+v", repo.requested)
	}
}

func TestServingApplicationLifecycleAcceptsDeploymentCompletion(t *testing.T) {
	app := lifecycleTestActionState().App
	task := Task{
		ID:         "task-1",
		ClusterID:  "cluster-1",
		Type:       platformtask.TaskTypeApplyDeployment,
		Status:     TaskStatusLeased,
		LeaseOwner: "agent-1",
		Payload: platformtask.EncodePayload(platformtask.RenderedDeploymentPayload{
			TypeValue:                 platformtask.TaskTypeApplyDeployment,
			ServingApplicationIDValue: app.ID,
			ResourceValue:             platformtask.ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
			EndpointValue:             platformtask.EndpointIntent{Name: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
			ManifestValues:            []platformtask.Manifest{{Name: "dgd.yaml", Content: "kind: DynamoGraphDeployment\n"}},
		}),
	}
	repo := &fakeLifecycleRepository{completionState: ServingApplicationCompletionState{App: app, Task: task}}
	lifecycle := NewServingApplicationLifecycle(repo)

	completed, err := lifecycle.AcceptTaskCompletion(context.Background(), "task-1", CompleteTaskRequest{AgentID: "agent-1", Status: TaskStatusSucceeded, Result: platformtask.EncodeResult(platformtask.DeploymentResult{TypeValue: platformtask.TaskTypeApplyDeployment, Resource: platformtask.ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"}, Phase: "Ready", EndpointURL: "http://ready.example/v1", HandledAt: time.Now()})}, "agent-1")
	if err != nil {
		t.Fatalf("accept completion: %v", err)
	}
	if completed.Status != TaskStatusSucceeded {
		t.Fatalf("unexpected completed task: %+v", completed)
	}
	if repo.accepted.Phase != ServingApplicationPhaseReady || repo.accepted.EndpointOperation != EndpointOperationUpsert || repo.accepted.Endpoint.URL != "http://ready.example/v1" {
		t.Fatalf("unexpected accepted completion: %+v", repo.accepted)
	}
}

type fakeLifecycleRepository struct {
	actionState     ServingApplicationActionState
	completionState ServingApplicationCompletionState
	requested       ServingApplicationRequestedAction
	accepted        ServingApplicationAcceptedCompletion
}

func (r *fakeLifecycleRepository) LoadActionState(string) (ServingApplicationActionState, error) {
	return r.actionState, nil
}

func (r *fakeLifecycleRepository) SaveRequestedAction(request ServingApplicationRequestedAction) (Task, error) {
	r.requested = request
	return Task{ID: "task-1", ClusterID: request.Task.ClusterID, Type: request.Task.Type, Status: TaskStatusPending, Payload: platformtask.EncodePayload(request.Task.Payload)}, nil
}

func (r *fakeLifecycleRepository) LoadCompletionState(string) (ServingApplicationCompletionState, error) {
	return r.completionState, nil
}

func (r *fakeLifecycleRepository) SaveAcceptedCompletion(accepted ServingApplicationAcceptedCompletion) (Task, error) {
	r.accepted = accepted
	return accepted.Task, nil
}

type fakeDeploymentRenderer struct{}

func (fakeDeploymentRenderer) Render(ServingRecipe, ServingApplication, ModelArtifact) (RenderedManifest, error) {
	return RenderedManifest{Name: "dgd.yaml", Content: "kind: DynamoGraphDeployment\n"}, nil
}

func lifecycleTestActionState() ServingApplicationActionState {
	return ServingApplicationActionState{
		App: ServingApplication{
			ID:        "app-1",
			ProjectID: "project-1",
			Name:      "DeepSeek V4 Flash",
			Model:     ModelIntent{ArtifactID: "artifact-1", Family: "deepseek-v4", Variant: "flash", Quantization: "fp8"},
			Placement: PlacementIntent{ClusterID: "cluster-1", Namespace: "dynamo-system"},
			Runtime:   RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "recipe-1"},
			Service:   ServiceIntent{EndpointName: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
		},
		Artifact: ModelArtifact{ID: "artifact-1"},
		Recipe:   ServingRecipe{},
		Cluster:  InferenceCluster{ID: "cluster-1"},
	}
}
