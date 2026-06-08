package management

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformtask "zhiliu/internal/task"
)

type ServingApplicationAction string

const (
	ServingApplicationActionPreview     ServingApplicationAction = "preview"
	ServingApplicationActionApply       ServingApplicationAction = "apply"
	ServingApplicationActionRedeploy    ServingApplicationAction = "redeploy"
	ServingApplicationActionRetire      ServingApplicationAction = "retire"
	ServingApplicationActionDiagnostics ServingApplicationAction = "diagnostics"
)

type ServingApplicationLifecycleRepository interface {
	LoadActionState(appID string) (ServingApplicationActionState, error)
	SaveRequestedAction(ServingApplicationRequestedAction) (Task, error)
	LoadCompletionState(taskID string) (ServingApplicationCompletionState, error)
	SaveAcceptedCompletion(ServingApplicationAcceptedCompletion) (Task, error)
}

type ServingApplicationActionState struct {
	App      ServingApplication
	Artifact ModelArtifact
	Recipe   ServingRecipe
	Cluster  InferenceCluster
}

type ServingApplicationRequestedAction struct {
	Task             platformtask.Envelope
	Actor            string
	TransitionPhase  ServingApplicationPhase
	TransitionReason string
}

type ServingApplicationCompletionState struct {
	App  ServingApplication
	Task Task
}

type ServingApplicationAcceptedCompletion struct {
	Task              Task
	Actor             string
	Phase             ServingApplicationPhase
	Reason            string
	EndpointOperation EndpointOperation
	Endpoint          EndpointRegistryEntry
}

type DeploymentRenderer interface {
	Render(recipe ServingRecipe, app ServingApplication, artifact ModelArtifact) (RenderedManifest, error)
}

type RecipeTemplateRenderer struct{}

func (RecipeTemplateRenderer) Render(recipe ServingRecipe, app ServingApplication, artifact ModelArtifact) (RenderedManifest, error) {
	return RenderRecipeTemplate(recipe, app, artifact)
}

// ServingApplicationLifecycle is the deep Management Plane Module for Serving
// Application task requests and task completions. Its Interface hides recipe
// rendering, task contract details, phase transitions, and Endpoint Registry
// updates behind two high-leverage entry points.
type ServingApplicationLifecycle struct {
	repo     ServingApplicationLifecycleRepository
	renderer DeploymentRenderer
	tasks    platformtask.Registry
	now      func() time.Time
}

func NewServingApplicationLifecycle(repo ServingApplicationLifecycleRepository) *ServingApplicationLifecycle {
	return NewServingApplicationLifecycleWithOptions(repo, RecipeTemplateRenderer{}, platformtask.DefaultRegistry())
}

func NewServingApplicationLifecycleWithOptions(repo ServingApplicationLifecycleRepository, renderer DeploymentRenderer, tasks platformtask.Registry) *ServingApplicationLifecycle {
	if renderer == nil {
		renderer = RecipeTemplateRenderer{}
	}
	return &ServingApplicationLifecycle{repo: repo, renderer: renderer, tasks: tasks, now: time.Now}
}

func (l *ServingApplicationLifecycle) RequestAction(_ context.Context, appID string, action ServingApplicationAction, actor string) (Task, error) {
	if l == nil || l.repo == nil {
		return Task{}, fmt.Errorf("%w: serving application lifecycle repository is required", ErrInvalidInput)
	}
	state, err := l.repo.LoadActionState(appID)
	if err != nil {
		return Task{}, err
	}
	request, err := l.planRequestedAction(state, action, actor)
	if err != nil {
		return Task{}, err
	}
	return l.repo.SaveRequestedAction(request)
}

func (l *ServingApplicationLifecycle) AcceptTaskCompletion(_ context.Context, taskID string, req CompleteTaskRequest, actor string) (Task, error) {
	if l == nil || l.repo == nil {
		return Task{}, fmt.Errorf("%w: serving application lifecycle repository is required", ErrInvalidInput)
	}
	state, err := l.repo.LoadCompletionState(taskID)
	if err != nil {
		return Task{}, err
	}
	accepted, err := l.planAcceptedCompletion(state, req, actor)
	if err != nil {
		return Task{}, err
	}
	return l.repo.SaveAcceptedCompletion(accepted)
}

func (l *ServingApplicationLifecycle) planRequestedAction(state ServingApplicationActionState, action ServingApplicationAction, actor string) (ServingApplicationRequestedAction, error) {
	app := state.App
	request := ServingApplicationRequestedAction{Actor: strings.TrimSpace(actor)}
	switch action {
	case ServingApplicationActionPreview, ServingApplicationActionApply, ServingApplicationActionRedeploy:
		manifest, err := l.renderer.Render(state.Recipe, app, state.Artifact)
		if err != nil {
			return ServingApplicationRequestedAction{}, err
		}
		taskType, phase, reason := renderedActionPlan(action)
		envelope, err := l.tasks.BuildRenderedDeployment(platformtask.RenderedDeploymentTaskInput{
			Type:                 taskType,
			ServingApplicationID: app.ID,
			ClusterID:            app.Placement.ClusterID,
			Resource:             platformtask.ResourceRef{Name: lifecycleResourceName(app), Namespace: app.Placement.Namespace},
			Endpoint:             platformtask.EndpointIntent{Name: app.Service.EndpointName, Protocol: app.Service.Protocol, Exposure: app.Service.Exposure},
			Manifests:            []platformtask.Manifest{{Name: manifest.Name, Content: manifest.Content}},
		})
		if err != nil {
			return ServingApplicationRequestedAction{}, err
		}
		request.Task = envelope
		request.TransitionPhase = phase
		request.TransitionReason = reason
		return request, nil
	case ServingApplicationActionRetire, ServingApplicationActionDiagnostics:
		taskType, phase, reason := resourceActionPlan(action)
		envelope, err := l.tasks.BuildResource(platformtask.ResourceTaskInput{
			Type:                 taskType,
			ServingApplicationID: app.ID,
			ClusterID:            app.Placement.ClusterID,
			Resource:             platformtask.ResourceRef{Name: lifecycleResourceName(app), Namespace: app.Placement.Namespace},
		})
		if err != nil {
			return ServingApplicationRequestedAction{}, err
		}
		request.Task = envelope
		request.TransitionPhase = phase
		request.TransitionReason = reason
		return request, nil
	default:
		return ServingApplicationRequestedAction{}, fmt.Errorf("%w: unsupported serving application action", ErrInvalidInput)
	}
}

func renderedActionPlan(action ServingApplicationAction) (platformtask.TaskType, ServingApplicationPhase, string) {
	switch action {
	case ServingApplicationActionApply:
		return platformtask.TaskTypeApplyDeployment, ServingApplicationPhaseApplying, "apply task created"
	case ServingApplicationActionRedeploy:
		return platformtask.TaskTypeDeleteBeforeApply, ServingApplicationPhaseApplying, "redeploy task created"
	default:
		return platformtask.TaskTypePreviewDeploymentDiff, "", ""
	}
}

func resourceActionPlan(action ServingApplicationAction) (platformtask.TaskType, ServingApplicationPhase, string) {
	switch action {
	case ServingApplicationActionRetire:
		return platformtask.TaskTypeRetireDeployment, ServingApplicationPhaseRetiring, "retire task created"
	default:
		return platformtask.TaskTypeFetchDiagnostics, "", ""
	}
}

func (l *ServingApplicationLifecycle) planAcceptedCompletion(state ServingApplicationCompletionState, req CompleteTaskRequest, actor string) (ServingApplicationAcceptedCompletion, error) {
	task := state.Task
	if req.Status != TaskStatusSucceeded && req.Status != TaskStatusFailed {
		return ServingApplicationAcceptedCompletion{}, fmt.Errorf("%w: completion status must be succeeded or failed", ErrInvalidInput)
	}
	if task.Status == TaskStatusSucceeded || task.Status == TaskStatusFailed {
		if task.LeaseOwner == req.AgentID && task.Status == req.Status {
			return ServingApplicationAcceptedCompletion{Task: task, Actor: actor}, nil
		}
		return ServingApplicationAcceptedCompletion{}, ErrTaskLeaseHeld
	}
	if task.LeaseOwner != req.AgentID {
		return ServingApplicationAcceptedCompletion{}, ErrTaskLeaseHeld
	}

	now := l.nowUTC()
	task.Status = req.Status
	task.Result = cloneAnyMap(req.Result)
	task.Error = strings.TrimSpace(req.Error)
	task.UpdatedAt = now

	accepted := ServingApplicationAcceptedCompletion{Task: task, Actor: strings.TrimSpace(actor)}
	phase, reason, endpointOperation, endpoint, err := l.completionEffect(state.App, task)
	if err != nil {
		accepted.Phase = ServingApplicationPhaseFailed
		accepted.Reason = err.Error()
		return accepted, nil
	}
	accepted.Phase = phase
	accepted.Reason = reason
	accepted.EndpointOperation = endpointOperation
	accepted.Endpoint = endpoint
	return accepted, nil
}

func (l *ServingApplicationLifecycle) completionEffect(app ServingApplication, task Task) (ServingApplicationPhase, string, EndpointOperation, EndpointRegistryEntry, error) {
	if task.Status == TaskStatusFailed {
		return ServingApplicationPhaseFailed, taskFailureReason(task), EndpointOperationNone, EndpointRegistryEntry{}, nil
	}
	if task.Status != TaskStatusSucceeded {
		return "", "", EndpointOperationNone, EndpointRegistryEntry{}, nil
	}

	effect := l.tasks.LifecycleEffectFor(task.Type)
	if effect == platformtask.LifecycleEffectNone || effect == platformtask.LifecycleEffectDiagnostics {
		return "", "", EndpointOperationNone, EndpointRegistryEntry{}, nil
	}

	result, err := l.tasks.DecodeResult(platformtask.NewDTO(task.ID, task.ClusterID, task.Type, task.Payload, task.Result, task.Error))
	if err != nil {
		return ServingApplicationPhaseFailed, err.Error(), EndpointOperationNone, EndpointRegistryEntry{}, nil
	}

	switch effect {
	case platformtask.LifecycleEffectPreview:
		return ServingApplicationPhaseValidated, "preview succeeded", EndpointOperationNone, EndpointRegistryEntry{}, nil
	case platformtask.LifecycleEffectDeployment:
		deployment, ok := result.(platformtask.DeploymentResult)
		if !ok {
			return "", "", EndpointOperationNone, EndpointRegistryEntry{}, fmt.Errorf("%w: expected deployment result", platformtask.ErrInvalidResult)
		}
		phase := ServingApplicationPhaseReady
		reason := "deployment ready"
		if strings.EqualFold(deployment.Phase, "failed") || strings.EqualFold(deployment.Phase, "error") {
			phase = ServingApplicationPhaseFailed
			reason = taskResultMessage(l.tasks, task)
		}
		if phase != ServingApplicationPhaseReady {
			return phase, reason, EndpointOperationNone, EndpointRegistryEntry{}, nil
		}
		return phase, reason, EndpointOperationUpsert, lifecycleReadyEndpoint(app, deployment.EndpointURL), nil
	case platformtask.LifecycleEffectRetire:
		return ServingApplicationPhaseRetired, "retire succeeded", EndpointOperationRemove, EndpointRegistryEntry{ServingApplicationID: app.ID}, nil
	default:
		return "", "", EndpointOperationNone, EndpointRegistryEntry{}, nil
	}
}

func (l *ServingApplicationLifecycle) nowUTC() time.Time {
	if l == nil || l.now == nil {
		return time.Now().UTC()
	}
	return l.now().UTC()
}

func lifecycleResourceName(app ServingApplication) string {
	resourceName := kubernetesName(app.Name)
	if resourceName == "" {
		resourceName = kubernetesName(app.ID)
	}
	return resourceName
}

func lifecycleEndpointName(app ServingApplication) string {
	endpointName := strings.TrimSpace(app.Service.EndpointName)
	if endpointName == "" {
		endpointName = lifecycleResourceName(app)
	}
	return endpointName
}

func lifecycleNamespace(app ServingApplication) string {
	namespace := strings.TrimSpace(app.Placement.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	return namespace
}

func lifecycleDefaultEndpointURL(app ServingApplication) string {
	return "http://" + lifecycleEndpointName(app) + "." + lifecycleNamespace(app) + ".svc.cluster.local:8000/v1"
}

func lifecycleReadyEndpoint(app ServingApplication, endpointURL string) EndpointRegistryEntry {
	endpointURL = strings.TrimSpace(endpointURL)
	if endpointURL == "" {
		endpointURL = lifecycleDefaultEndpointURL(app)
	}
	return EndpointRegistryEntry{
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Namespace:            lifecycleNamespace(app),
		EndpointName:         lifecycleEndpointName(app),
		URL:                  endpointURL,
		Ready:                true,
	}
}
