package management

import (
	"testing"

	platformtask "zhiliu/internal/task"
)

func TestServingApplicationControlLoopPlansApplyTask(t *testing.T) {
	app := controlLoopTestApp()
	plan, err := (ServingApplicationControlLoop{}).PlanRenderedTask(app, TaskTypeApplyDeployment, RenderedDeploymentManifest{Name: "dgd.yaml", Content: "kind: DynamoGraphDeployment\n"})
	if err != nil {
		t.Fatalf("plan rendered task: %v", err)
	}
	if plan.ClusterID != app.Placement.ClusterID || plan.Type != TaskTypeApplyDeployment {
		t.Fatalf("unexpected task plan: %+v", plan)
	}
	if plan.TransitionPhase != ServingApplicationPhaseApplying || plan.TransitionReason != "apply task created" {
		t.Fatalf("unexpected transition plan: %+v", plan)
	}
	payload, ok := plan.Payload.(platformtask.RenderedDeploymentPayload)
	if !ok || payload.ServingApplicationID() != app.ID || payload.Resource().Namespace != app.Placement.Namespace || len(payload.Manifests()) != 1 {
		t.Fatalf("unexpected typed payload: %+v", plan.Payload)
	}
}

func TestServingApplicationControlLoopCompletesDeploymentTask(t *testing.T) {
	app := controlLoopTestApp()
	control := ServingApplicationControlLoop{}
	plan, err := control.PlanRenderedTask(app, TaskTypeApplyDeployment, RenderedDeploymentManifest{Name: "dgd.yaml", Content: "kind: DynamoGraphDeployment\n"})
	if err != nil {
		t.Fatalf("plan rendered task: %v", err)
	}
	task := Task{
		ID:        "task-1",
		ClusterID: plan.ClusterID,
		Type:      plan.Type,
		Payload:   platformtask.EncodePayload(plan.Payload),
		Status:    TaskStatusSucceeded,
		Result: platformtask.EncodeResult(platformtask.DeploymentResult{
			TypeValue:   platformtask.TypeApplyDeployment,
			Resource:    platformtask.ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
			EndpointURL: "http://deepseek-v4-flash.dynamo-system.svc.cluster.local:8000/v1",
			Phase:       "Ready",
			Message:     "ok",
		}),
	}
	completion, err := control.CompleteTask(app, task)
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if completion.Phase != ServingApplicationPhaseReady || !completion.UpsertEndpoint || completion.EndpointURL == "" {
		t.Fatalf("unexpected completion plan: %+v", completion)
	}
}

func TestServingApplicationControlLoopCompletesRetireTask(t *testing.T) {
	app := controlLoopTestApp()
	control := ServingApplicationControlLoop{}
	plan, err := control.PlanResourceTask(app, TaskTypeRetireDeployment, "deepseek-v4-flash")
	if err != nil {
		t.Fatalf("plan retire task: %v", err)
	}
	task := Task{
		ID:        "task-1",
		ClusterID: plan.ClusterID,
		Type:      plan.Type,
		Payload:   platformtask.EncodePayload(plan.Payload),
		Status:    TaskStatusSucceeded,
		Result:    platformtask.EncodeResult(platformtask.RetireResult{Resource: platformtask.ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"}, Deleted: true}),
	}
	completion, err := control.CompleteTask(app, task)
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if completion.Phase != ServingApplicationPhaseRetired || !completion.RemoveEndpoint {
		t.Fatalf("unexpected completion plan: %+v", completion)
	}
}

func controlLoopTestApp() ServingApplication {
	return ServingApplication{
		ID:        "app-1",
		ProjectID: "project-1",
		Name:      "DeepSeek V4 Flash",
		Placement: PlacementIntent{ClusterID: "cluster-1", Namespace: "dynamo-system"},
		Service:   ServiceIntent{EndpointName: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
	}
}
