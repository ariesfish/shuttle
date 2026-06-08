package management

import (
	"strings"

	platformtask "zhiliu/internal/task"
)

type ServingApplicationControlLoop struct{}

type RenderedDeploymentManifest struct {
	Name    string
	Content string
}

type ServingApplicationTaskPlan struct {
	ClusterID        string
	Type             TaskType
	Payload          platformtask.Payload
	TransitionPhase  ServingApplicationPhase
	TransitionReason string
}

type ServingApplicationCompletionPlan struct {
	Phase          ServingApplicationPhase
	Reason         string
	EndpointURL    string
	UpsertEndpoint bool
	RemoveEndpoint bool
}

func (ServingApplicationControlLoop) PlanRenderedTask(app ServingApplication, taskType TaskType, manifest RenderedDeploymentManifest) (ServingApplicationTaskPlan, error) {
	envelope, err := platformtask.BuildRenderedDeploymentTask(platformtask.RenderedDeploymentTaskInput{
		Type:                 platformtask.Type(taskType),
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Resource:             platformtask.ResourceRef{Name: kubernetesName(app.Name), Namespace: app.Placement.Namespace},
		Endpoint:             platformtask.EndpointIntent{Name: app.Service.EndpointName, Protocol: app.Service.Protocol, Exposure: app.Service.Exposure},
		Manifests:            []platformtask.Manifest{{Name: manifest.Name, Content: manifest.Content}},
	})
	if err != nil {
		return ServingApplicationTaskPlan{}, err
	}
	plan := ServingApplicationTaskPlan{ClusterID: envelope.ClusterID, Type: TaskType(envelope.Type), Payload: envelope.Payload}
	switch taskType {
	case TaskTypeApplyDeployment:
		plan.TransitionPhase = ServingApplicationPhaseApplying
		plan.TransitionReason = "apply task created"
	case TaskTypeDeleteBeforeApply:
		plan.TransitionPhase = ServingApplicationPhaseApplying
		plan.TransitionReason = "redeploy task created"
	}
	return plan, nil
}

func (ServingApplicationControlLoop) PlanResourceTask(app ServingApplication, taskType TaskType, resourceName string) (ServingApplicationTaskPlan, error) {
	envelope, err := platformtask.BuildResourceTask(platformtask.ResourceTaskInput{
		Type:                 platformtask.Type(taskType),
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Resource:             platformtask.ResourceRef{Name: resourceName, Namespace: app.Placement.Namespace},
	})
	if err != nil {
		return ServingApplicationTaskPlan{}, err
	}
	plan := ServingApplicationTaskPlan{ClusterID: envelope.ClusterID, Type: TaskType(envelope.Type), Payload: envelope.Payload}
	if taskType == TaskTypeRetireDeployment {
		plan.TransitionPhase = ServingApplicationPhaseRetiring
		plan.TransitionReason = "retire task created"
	}
	return plan, nil
}

func (ServingApplicationControlLoop) ServingApplicationIDForTask(task Task) (string, error) {
	payload, err := platformtask.DecodePayload(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload})
	if err != nil {
		return "", err
	}
	return payload.ServingApplicationID(), nil
}

func taskFailureReason(task Task) string {
	if strings.TrimSpace(task.Error) != "" {
		return task.Error
	}
	return taskResultMessage(task)
}

func taskResultMessage(task Task) string {
	result, err := platformtask.DecodeResult(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload, Result: task.Result, Error: task.Error})
	if err == nil {
		switch value := result.(type) {
		case platformtask.DeploymentResult:
			if strings.TrimSpace(value.Message) != "" {
				return strings.TrimSpace(value.Message)
			}
			if strings.TrimSpace(value.Phase) != "" {
				return "task result phase: " + strings.TrimSpace(value.Phase)
			}
		case platformtask.RetireResult:
			if strings.TrimSpace(value.Message) != "" {
				return strings.TrimSpace(value.Message)
			}
		}
	}
	return string(task.Type) + " completed"
}

func (ServingApplicationControlLoop) CompleteTask(app ServingApplication, task Task) (ServingApplicationCompletionPlan, error) {
	if task.Status == TaskStatusFailed {
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseFailed, Reason: taskFailureReason(task)}, nil
	}
	if task.Status != TaskStatusSucceeded {
		return ServingApplicationCompletionPlan{}, nil
	}

	result, err := platformtask.DecodeResult(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload, Result: task.Result, Error: task.Error})
	if err != nil {
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseFailed, Reason: err.Error()}, nil
	}

	switch value := result.(type) {
	case platformtask.PreviewResult:
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseValidated, Reason: "preview succeeded"}, nil
	case platformtask.DeploymentResult:
		phase := ServingApplicationPhaseReady
		reason := "deployment ready"
		if strings.EqualFold(value.Phase, "failed") || strings.EqualFold(value.Phase, "error") {
			phase = ServingApplicationPhaseFailed
			reason = taskResultMessage(task)
		}
		return ServingApplicationCompletionPlan{Phase: phase, Reason: reason, EndpointURL: strings.TrimSpace(value.EndpointURL), UpsertEndpoint: phase == ServingApplicationPhaseReady}, nil
	case platformtask.RetireResult:
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseRetired, Reason: "retire succeeded", RemoveEndpoint: true}, nil
	default:
		return ServingApplicationCompletionPlan{}, nil
	}
}
