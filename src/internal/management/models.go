package management

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type InferenceCluster struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ClusterAgent struct {
	ID            string            `json:"id"`
	ClusterID     string            `json:"clusterId"`
	Version       string            `json:"version,omitempty"`
	Capabilities  map[string]string `json:"capabilities,omitempty"`
	LastHeartbeat time.Time         `json:"lastHeartbeat,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type ModelArtifact struct {
	ID            string    `json:"id"`
	Family        string    `json:"family"`
	Variant       string    `json:"variant"`
	Revision      string    `json:"revision"`
	PVCName       string    `json:"pvcName,omitempty"`
	PVCMountPath  string    `json:"pvcMountPath"`
	PVCModelPath  string    `json:"pvcModelPath"`
	HostCachePath string    `json:"hostCachePath,omitempty"`
	Quantization  string    `json:"quantization"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ServingApplicationPhase string

const (
	ServingApplicationPhaseDraft           ServingApplicationPhase = "Draft"
	ServingApplicationPhaseValidated       ServingApplicationPhase = "Validated"
	ServingApplicationPhasePendingApproval ServingApplicationPhase = "PendingApproval"
	ServingApplicationPhaseApplying        ServingApplicationPhase = "Applying"
	ServingApplicationPhaseDeploying       ServingApplicationPhase = "Deploying"
	ServingApplicationPhaseReady           ServingApplicationPhase = "Ready"
	ServingApplicationPhaseFailed          ServingApplicationPhase = "Failed"
	ServingApplicationPhaseRetiring        ServingApplicationPhase = "Retiring"
	ServingApplicationPhaseRetired         ServingApplicationPhase = "Retired"
)

type ServingApplication struct {
	ID            string                  `json:"id"`
	ProjectID     string                  `json:"projectId"`
	Name          string                  `json:"name"`
	Model         ModelIntent             `json:"model"`
	Placement     PlacementIntent         `json:"placement"`
	Runtime       RuntimeIntent           `json:"runtime"`
	Service       ServiceIntent           `json:"service"`
	Optimization  OptimizationIntent      `json:"optimization"`
	DesiredState  string                  `json:"desiredState"`
	Phase         ServingApplicationPhase `json:"phase"`
	ActiveVersion int                     `json:"activeVersion"`
	EndpointURL   string                  `json:"endpointUrl,omitempty"`
	GrafanaURL    string                  `json:"grafanaUrl,omitempty"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
}

type ModelIntent struct {
	Family       string `json:"family"`
	Variant      string `json:"variant"`
	ArtifactID   string `json:"artifactId"`
	Quantization string `json:"quantization"`
}

type PlacementIntent struct {
	ClusterID         string `json:"clusterId"`
	AcceleratorPoolID string `json:"acceleratorPoolId,omitempty"`
	Namespace         string `json:"namespace"`
}

type RuntimeIntent struct {
	Backend  string         `json:"backend"`
	Topology string         `json:"topology"`
	Recipe   string         `json:"recipe"`
	Replicas map[string]int `json:"replicas,omitempty"`
}

type ServiceIntent struct {
	EndpointName string `json:"endpointName"`
	Protocol     string `json:"protocol"`
	Exposure     string `json:"exposure"`
}

type OptimizationIntent struct {
	Target        string   `json:"target"`
	TTFTMS        *float64 `json:"ttftMs,omitempty"`
	ITLMS         *float64 `json:"itlMs,omitempty"`
	ProfilingMode string   `json:"profilingMode"`
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusLeased    TaskStatus = "leased"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type TaskType string

const (
	TaskTypeRegisterCluster       TaskType = "RegisterCluster"
	TaskTypeValidateIntent        TaskType = "ValidateIntent"
	TaskTypePreviewDeploymentDiff TaskType = "PreviewDeploymentDiff"
	TaskTypeApplyDeployment       TaskType = "ApplyDeployment"
	TaskTypeDeleteBeforeApply     TaskType = "DeleteBeforeApplyRedeploy"
	TaskTypeInspectStatus         TaskType = "InspectDeploymentStatus"
	TaskTypeRetireDeployment      TaskType = "RetireDeployment"
	TaskTypeFetchDiagnostics      TaskType = "FetchDiagnostics"
	TaskTypeSyncEndpointReadiness TaskType = "SyncEndpointReadiness"
)

type Task struct {
	ID             string         `json:"id"`
	ClusterID      string         `json:"clusterId"`
	Type           TaskType       `json:"type"`
	Status         TaskStatus     `json:"status"`
	Payload        map[string]any `json:"payload,omitempty"`
	LeaseOwner     string         `json:"leaseOwner,omitempty"`
	LeaseExpiresAt time.Time      `json:"leaseExpiresAt,omitempty"`
	Result         map[string]any `json:"result,omitempty"`
	Error          string         `json:"error,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type CreateClusterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type RegisterAgentRequest struct {
	ClusterID    string            `json:"clusterId"`
	Version      string            `json:"version,omitempty"`
	Capabilities map[string]string `json:"capabilities,omitempty"`
}

type HeartbeatRequest struct {
	Version      string            `json:"version,omitempty"`
	Capabilities map[string]string `json:"capabilities,omitempty"`
}

type CreateModelArtifactRequest struct {
	Family        string `json:"family"`
	Variant       string `json:"variant"`
	Revision      string `json:"revision"`
	PVCName       string `json:"pvcName,omitempty"`
	PVCMountPath  string `json:"pvcMountPath"`
	PVCModelPath  string `json:"pvcModelPath"`
	HostCachePath string `json:"hostCachePath,omitempty"`
	Quantization  string `json:"quantization"`
}

type CreateServingApplicationRequest struct {
	ProjectID    string             `json:"projectId"`
	Name         string             `json:"name"`
	Model        ModelIntent        `json:"model"`
	Placement    PlacementIntent    `json:"placement"`
	Runtime      RuntimeIntent      `json:"runtime"`
	Service      ServiceIntent      `json:"service"`
	Optimization OptimizationIntent `json:"optimization"`
}

type CreateTaskRequest struct {
	ClusterID string         `json:"clusterId"`
	Type      TaskType       `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type CreatePreviewTaskRequest struct {
	ServingApplicationID string `json:"servingApplicationId"`
}

type CreateApplyTaskRequest struct {
	ServingApplicationID string `json:"servingApplicationId"`
}

type CreateRedeployTaskRequest struct {
	ServingApplicationID string `json:"servingApplicationId"`
}

type CreateRetireTaskRequest struct {
	ServingApplicationID string `json:"servingApplicationId"`
}

type LeaseTaskRequest struct {
	AgentID string `json:"agentId"`
}

type CompleteTaskRequest struct {
	AgentID string         `json:"agentId"`
	Status  TaskStatus     `json:"status"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}
