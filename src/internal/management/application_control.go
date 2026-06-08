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
	Type             platformtask.TaskType
	Payload          platformtask.Payload
	TransitionPhase  ServingApplicationPhase
	TransitionReason string
}

type EndpointOperation string

const (
	EndpointOperationNone   EndpointOperation = ""
	EndpointOperationUpsert EndpointOperation = "upsert"
	EndpointOperationRemove EndpointOperation = "remove"
)

type ServingApplicationCompletionPlan struct {
	Phase             ServingApplicationPhase
	Reason            string
	EndpointOperation EndpointOperation
	Endpoint          EndpointRegistryEntry
}

func (c ServingApplicationControlLoop) PlanPreviewTask(app ServingApplication, manifest RenderedDeploymentManifest) (ServingApplicationTaskPlan, error) {
	return c.PlanRenderedTask(app, platformtask.TaskTypePreviewDeploymentDiff, manifest)
}

func (c ServingApplicationControlLoop) PlanApplyTask(app ServingApplication, manifest RenderedDeploymentManifest) (ServingApplicationTaskPlan, error) {
	return c.PlanRenderedTask(app, platformtask.TaskTypeApplyDeployment, manifest)
}

func (c ServingApplicationControlLoop) PlanRedeployTask(app ServingApplication, manifest RenderedDeploymentManifest) (ServingApplicationTaskPlan, error) {
	return c.PlanRenderedTask(app, platformtask.TaskTypeDeleteBeforeApply, manifest)
}

func (c ServingApplicationControlLoop) PlanRetireTask(app ServingApplication) (ServingApplicationTaskPlan, error) {
	return c.PlanResourceTask(app, platformtask.TaskTypeRetireDeployment, c.resourceName(app))
}

func (c ServingApplicationControlLoop) PlanDiagnosticsTask(app ServingApplication) (ServingApplicationTaskPlan, error) {
	return c.PlanResourceTask(app, platformtask.TaskTypeFetchDiagnostics, c.resourceName(app))
}

func (ServingApplicationControlLoop) PlanRenderedTask(app ServingApplication, taskType platformtask.TaskType, manifest RenderedDeploymentManifest) (ServingApplicationTaskPlan, error) {
	envelope, err := platformtask.BuildRenderedDeploymentTask(platformtask.RenderedDeploymentTaskInput{
		Type:                 platformtask.TaskType(taskType),
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Resource:             platformtask.ResourceRef{Name: kubernetesName(app.Name), Namespace: app.Placement.Namespace},
		Endpoint:             platformtask.EndpointIntent{Name: app.Service.EndpointName, Protocol: app.Service.Protocol, Exposure: app.Service.Exposure},
		Manifests:            []platformtask.Manifest{{Name: manifest.Name, Content: manifest.Content}},
	})
	if err != nil {
		return ServingApplicationTaskPlan{}, err
	}
	plan := ServingApplicationTaskPlan{ClusterID: envelope.ClusterID, Type: platformtask.TaskType(envelope.Type), Payload: envelope.Payload}
	switch taskType {
	case platformtask.TaskTypeApplyDeployment:
		plan.TransitionPhase = ServingApplicationPhaseApplying
		plan.TransitionReason = "apply task created"
	case platformtask.TaskTypeDeleteBeforeApply:
		plan.TransitionPhase = ServingApplicationPhaseApplying
		plan.TransitionReason = "redeploy task created"
	}
	return plan, nil
}

func (ServingApplicationControlLoop) PlanResourceTask(app ServingApplication, taskType platformtask.TaskType, resourceName string) (ServingApplicationTaskPlan, error) {
	envelope, err := platformtask.BuildResourceTask(platformtask.ResourceTaskInput{
		Type:                 platformtask.TaskType(taskType),
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Resource:             platformtask.ResourceRef{Name: resourceName, Namespace: app.Placement.Namespace},
	})
	if err != nil {
		return ServingApplicationTaskPlan{}, err
	}
	plan := ServingApplicationTaskPlan{ClusterID: envelope.ClusterID, Type: platformtask.TaskType(envelope.Type), Payload: envelope.Payload}
	if taskType == platformtask.TaskTypeRetireDeployment {
		plan.TransitionPhase = ServingApplicationPhaseRetiring
		plan.TransitionReason = "retire task created"
	}
	return plan, nil
}

func (ServingApplicationControlLoop) ServingApplicationIDForTask(task Task) (string, error) {
	payload, err := platformtask.DecodePayload(platformtask.NewDTO(task.ID, task.ClusterID, platformtask.TaskType(task.Type), task.Payload, nil, ""))
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
	result, err := platformtask.DecodeResult(platformtask.NewDTO(task.ID, task.ClusterID, platformtask.TaskType(task.Type), task.Payload, task.Result, task.Error))
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

func (c ServingApplicationControlLoop) resourceName(app ServingApplication) string {
	resourceName := kubernetesName(app.Name)
	if resourceName == "" {
		resourceName = kubernetesName(app.ID)
	}
	return resourceName
}

func (c ServingApplicationControlLoop) endpointName(app ServingApplication) string {
	endpointName := strings.TrimSpace(app.Service.EndpointName)
	if endpointName == "" {
		endpointName = c.resourceName(app)
	}
	return endpointName
}

func (c ServingApplicationControlLoop) namespace(app ServingApplication) string {
	namespace := strings.TrimSpace(app.Placement.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

func (c ServingApplicationControlLoop) defaultEndpointURL(app ServingApplication) string {
	return "http://" + c.endpointName(app) + "." + c.namespace(app) + ".svc.cluster.local:8000/v1"
}

func (c ServingApplicationControlLoop) ReadyEndpoint(app ServingApplication, endpointURL string) EndpointRegistryEntry {
	endpointURL = strings.TrimSpace(endpointURL)
	if endpointURL == "" {
		endpointURL = c.defaultEndpointURL(app)
	}
	return EndpointRegistryEntry{
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Namespace:            c.namespace(app),
		EndpointName:         c.endpointName(app),
		URL:                  endpointURL,
		Ready:                true,
	}
}

func (c ServingApplicationControlLoop) CompleteTask(app ServingApplication, task Task) (ServingApplicationCompletionPlan, error) {
	if task.Status == TaskStatusFailed {
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseFailed, Reason: taskFailureReason(task)}, nil
	}
	if task.Status != TaskStatusSucceeded {
		return ServingApplicationCompletionPlan{}, nil
	}

	result, err := platformtask.DecodeResult(platformtask.NewDTO(task.ID, task.ClusterID, platformtask.TaskType(task.Type), task.Payload, task.Result, task.Error))
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
		completion := ServingApplicationCompletionPlan{Phase: phase, Reason: reason}
		if phase == ServingApplicationPhaseReady {
			completion.EndpointOperation = EndpointOperationUpsert
			completion.Endpoint = c.ReadyEndpoint(app, value.EndpointURL)
		}
		return completion, nil
	case platformtask.RetireResult:
		return ServingApplicationCompletionPlan{Phase: ServingApplicationPhaseRetired, Reason: "retire succeeded", EndpointOperation: EndpointOperationRemove, Endpoint: EndpointRegistryEntry{ServingApplicationID: app.ID}}, nil
	default:
		return ServingApplicationCompletionPlan{}, nil
	}
}
