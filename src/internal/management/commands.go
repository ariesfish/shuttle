package management

import (
	"context"
	"errors"
	"log/slog"
)

var ErrForbidden = errors.New("forbidden")

type ManagementCommands struct {
	store     ManagementStore
	lifecycle *ServingApplicationLifecycle
	logger    *slog.Logger
}

func NewManagementCommands(store ManagementStore, logger *slog.Logger) *ManagementCommands {
	if logger == nil {
		logger = slog.Default()
	}
	return &ManagementCommands{store: store, lifecycle: NewServingApplicationLifecycle(store), logger: logger}
}

func (c *ManagementCommands) CreateProject(ctx context.Context, req CreateProjectRequest) (Project, error) {
	if err := requireActorRole(ctx, "admin"); err != nil {
		return Project{}, err
	}
	project, err := c.store.CreateProject(req)
	if err == nil {
		c.recordAudit(ctx, "create_project", project.ID, map[string]any{"name": project.Name})
	}
	return project, err
}

func (c *ManagementCommands) CreateCluster(ctx context.Context, req CreateClusterRequest) (InferenceCluster, error) {
	if err := requireActorRole(ctx, "admin"); err != nil {
		return InferenceCluster{}, err
	}
	cluster, err := c.store.CreateCluster(req)
	if err == nil {
		c.recordAudit(ctx, "create_cluster", cluster.ID, map[string]any{"name": cluster.Name})
	}
	return cluster, err
}

func (c *ManagementCommands) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (ClusterAgent, error) {
	if err := requireActorRole(ctx, "admin", "agent"); err != nil {
		return ClusterAgent{}, err
	}
	agent, err := c.store.RegisterAgent(req)
	if err == nil {
		c.recordAudit(ctx, "register_agent", agent.ID, map[string]any{"clusterId": agent.ClusterID})
	}
	return agent, err
}

func (c *ManagementCommands) CreateModelArtifact(ctx context.Context, req CreateModelArtifactRequest) (ModelArtifact, error) {
	if err := requireActorRole(ctx, "admin", "operator"); err != nil {
		return ModelArtifact{}, err
	}
	artifact, err := c.store.CreateModelArtifact(req)
	if err == nil {
		c.recordAudit(ctx, "create_model_artifact", artifact.ID, map[string]any{"family": artifact.Family, "variant": artifact.Variant})
	}
	return artifact, err
}

func (c *ManagementCommands) CreateServingApplication(ctx context.Context, req CreateServingApplicationRequest) (ServingApplication, error) {
	if err := requireActorRole(ctx, "admin", "operator"); err != nil {
		return ServingApplication{}, err
	}
	app, err := c.store.CreateServingApplication(req)
	if err == nil {
		c.recordAudit(ctx, "create_serving_application", app.ID, map[string]any{"projectId": app.ProjectID, "name": app.Name})
	}
	return app, err
}

func (c *ManagementCommands) CreatePreviewTask(ctx context.Context, appID string) (Task, error) {
	return c.requestServingApplicationAction(ctx, "create_preview_task", appID, ServingApplicationActionPreview)
}

func (c *ManagementCommands) CreateApplyTask(ctx context.Context, appID string) (Task, error) {
	return c.requestServingApplicationAction(ctx, "create_apply_task", appID, ServingApplicationActionApply)
}

func (c *ManagementCommands) CreateRedeployTask(ctx context.Context, appID string) (Task, error) {
	return c.requestServingApplicationAction(ctx, "create_redeploy_task", appID, ServingApplicationActionRedeploy)
}

func (c *ManagementCommands) CreateRetireTask(ctx context.Context, appID string) (Task, error) {
	return c.requestServingApplicationAction(ctx, "create_retire_task", appID, ServingApplicationActionRetire)
}

func (c *ManagementCommands) CreateDiagnosticsTask(ctx context.Context, appID string) (Task, error) {
	return c.requestServingApplicationAction(ctx, "create_diagnostics_task", appID, ServingApplicationActionDiagnostics)
}

func (c *ManagementCommands) CreateTask(ctx context.Context, req CreateTaskRequest) (Task, error) {
	if err := requireActorRole(ctx, "admin", "operator"); err != nil {
		return Task{}, err
	}
	task, err := c.store.CreateTask(req)
	if err == nil {
		c.recordAudit(ctx, "create_task", task.ID, map[string]any{"type": task.Type, "clusterId": task.ClusterID})
	}
	return task, err
}

func (c *ManagementCommands) CompleteTask(ctx context.Context, taskID string, req CompleteTaskRequest) (Task, error) {
	if err := requireActorRole(ctx, "admin", "operator", "agent"); err != nil {
		return Task{}, err
	}
	task, err := c.lifecycle.AcceptTaskCompletion(ctx, taskID, req, ActorFromContext(ctx).Name)
	if err == nil {
		c.recordAudit(ctx, "complete_task", task.ID, map[string]any{"status": task.Status, "type": task.Type})
	}
	return task, err
}

func (c *ManagementCommands) requestServingApplicationAction(ctx context.Context, auditAction string, appID string, action ServingApplicationAction) (Task, error) {
	if err := requireActorRole(ctx, "admin", "operator"); err != nil {
		return Task{}, err
	}
	task, err := c.lifecycle.RequestAction(ctx, appID, action, ActorFromContext(ctx).Name)
	if err == nil {
		c.recordAudit(ctx, auditAction, task.ID, map[string]any{"servingApplicationId": appID})
	}
	return task, err
}

func (c *ManagementCommands) recordAudit(ctx context.Context, action string, resource string, metadata map[string]any) {
	actor := ActorFromContext(ctx)
	if _, err := c.store.RecordAudit(actor.Name, action, resource, metadata); err != nil {
		c.logger.Error("record audit", "error", err, "action", action, "resource", resource)
	}
}

func requireActorRole(ctx context.Context, roles ...string) error {
	actor := ActorFromContext(ctx)
	for _, role := range roles {
		if actor.Role == role {
			return nil
		}
	}
	return ErrForbidden
}
