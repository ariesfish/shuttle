package management

import (
	"time"

	platformtask "zhiliu/internal/task"
)

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AuditRecord struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type InferenceCluster struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	PrometheusURL string    `json:"prometheusUrl,omitempty"`
	GrafanaURL    string    `json:"grafanaUrl,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ClusterAgent struct {
	ID                      string            `json:"id"`
	ClusterID               string            `json:"clusterId"`
	Version                 string            `json:"version,omitempty"`
	Capabilities            map[string]string `json:"capabilities,omitempty"`
	LastInventoryRevision   string            `json:"lastInventoryRevision,omitempty"`
	LastInventoryFreshness  string            `json:"lastInventoryFreshness,omitempty"`
	LastInventoryObservedAt time.Time         `json:"lastInventoryObservedAt,omitempty"`
	LastInventoryReportedAt time.Time         `json:"lastInventoryReportedAt,omitempty"`
	LastHeartbeat           time.Time         `json:"lastHeartbeat,omitempty"`
	CreatedAt               time.Time         `json:"createdAt"`
	UpdatedAt               time.Time         `json:"updatedAt"`
}

type AcceleratorInventoryFreshness string

const (
	AcceleratorInventoryFreshnessFresh       AcceleratorInventoryFreshness = "fresh"
	AcceleratorInventoryFreshnessStale       AcceleratorInventoryFreshness = "stale"
	AcceleratorInventoryFreshnessMissing     AcceleratorInventoryFreshness = "missing"
	AcceleratorInventoryFreshnessUnsupported AcceleratorInventoryFreshness = "unsupported"
)

type AcceleratorInventory struct {
	ClusterID          string                        `json:"clusterId"`
	AgentID            string                        `json:"agentId"`
	SchemaVersion      string                        `json:"schemaVersion"`
	Revision           string                        `json:"revision"`
	ObservedAt         time.Time                     `json:"observedAt"`
	ReportedAt         time.Time                     `json:"reportedAt"`
	Freshness          AcceleratorInventoryFreshness `json:"freshness"`
	RevisionCount      int                           `json:"revisionCount,omitempty"`
	Nodes              []AcceleratorInventoryNode    `json:"nodes"`
	ProbeStatuses      []AcceleratorInventoryProbe   `json:"probeStatuses,omitempty"`
	CollectionMetadata map[string]string             `json:"collectionMetadata,omitempty"`
}

type AcceleratorInventoryNode struct {
	Name                     string                             `json:"name"`
	Labels                   map[string]string                  `json:"labels,omitempty"`
	Taints                   []string                           `json:"taints,omitempty"`
	Capacity                 map[string]string                  `json:"capacity,omitempty"`
	Allocatable              map[string]string                  `json:"allocatable,omitempty"`
	AcceleratorResourceNames []string                           `json:"acceleratorResourceNames,omitempty"`
	Accelerators             []AcceleratorInventoryAccelerator  `json:"accelerators,omitempty"`
	Connectivity             []AcceleratorInventoryConnectivity `json:"connectivity,omitempty"`
	ObservedAt               time.Time                          `json:"observedAt"`
}

type AcceleratorInventoryConnectivity struct {
	Type       string            `json:"type"`
	Present    bool              `json:"present"`
	Confidence string            `json:"confidence"`
	Summary    string            `json:"summary,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

type AcceleratorInventoryAccelerator struct {
	Vendor        string            `json:"vendor"`
	Product       string            `json:"product,omitempty"`
	DeviceCount   int               `json:"deviceCount,omitempty"`
	MemoryMiB     int               `json:"memoryMiB,omitempty"`
	VendorDetails map[string]string `json:"vendorDetails,omitempty"`
}

type AcceleratorInventoryProbe struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ReportAcceleratorInventoryRequest struct {
	AgentID            string                      `json:"agentId"`
	SchemaVersion      string                      `json:"schemaVersion"`
	Revision           string                      `json:"revision,omitempty"`
	ObservedAt         time.Time                   `json:"observedAt"`
	Nodes              []AcceleratorInventoryNode  `json:"nodes"`
	ProbeStatuses      []AcceleratorInventoryProbe `json:"probeStatuses,omitempty"`
	CollectionMetadata map[string]string           `json:"collectionMetadata,omitempty"`
}

type AcceleratorPool struct {
	ID           string            `json:"id"`
	ClusterID    string            `json:"clusterId"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type AcceleratorPoolSummary struct {
	Pool              AcceleratorPool               `json:"pool"`
	Freshness         AcceleratorInventoryFreshness `json:"freshness"`
	InventoryRevision string                        `json:"inventoryRevision,omitempty"`
	NodeCount         int                           `json:"nodeCount"`
	AcceleratorCount  int                           `json:"acceleratorCount"`
	AcceleratorModels map[string]int                `json:"acceleratorModels,omitempty"`
	MemoryMiBSummary  map[string]int                `json:"memoryMiBSummary,omitempty"`
	Labels            map[string][]string           `json:"labels,omitempty"`
	Taints            []string                      `json:"taints,omitempty"`
	Warnings          []string                      `json:"warnings,omitempty"`
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

type ServingApplicationTransition struct {
	ID                   string                  `json:"id"`
	ServingApplicationID string                  `json:"servingApplicationId"`
	Actor                string                  `json:"actor"`
	TaskID               string                  `json:"taskId,omitempty"`
	From                 ServingApplicationPhase `json:"from,omitempty"`
	To                   ServingApplicationPhase `json:"to"`
	Reason               string                  `json:"reason,omitempty"`
	CreatedAt            time.Time               `json:"createdAt"`
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

type ObservabilityEntry struct {
	ServingApplicationID string            `json:"servingApplicationId"`
	ClusterID            string            `json:"clusterId"`
	Namespace            string            `json:"namespace"`
	GrafanaURL           string            `json:"grafanaUrl,omitempty"`
	PrometheusURL        string            `json:"prometheusUrl,omitempty"`
	PrometheusQueries    []PrometheusQuery `json:"prometheusQueries"`
}

type PrometheusQuery struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
}

type PrometheusQueryResult struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Query       string    `json:"query"`
	Value       string    `json:"value,omitempty"`
	Error       string    `json:"error,omitempty"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

type ObservabilitySummary struct {
	ServingApplicationID string                  `json:"servingApplicationId"`
	ClusterID            string                  `json:"clusterId"`
	Namespace            string                  `json:"namespace"`
	PrometheusURL        string                  `json:"prometheusUrl,omitempty"`
	Results              []PrometheusQueryResult `json:"results"`
}

type EndpointRegistryEntry struct {
	ID                   string    `json:"id"`
	ServingApplicationID string    `json:"servingApplicationId"`
	ClusterID            string    `json:"clusterId"`
	Namespace            string    `json:"namespace"`
	EndpointName         string    `json:"endpointName"`
	URL                  string    `json:"url"`
	Ready                bool      `json:"ready"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusLeased    TaskStatus = "leased"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID             string                `json:"id"`
	ClusterID      string                `json:"clusterId"`
	Type           platformtask.TaskType `json:"type"`
	Status         TaskStatus            `json:"status"`
	Payload        map[string]any        `json:"payload,omitempty"`
	LeaseOwner     string                `json:"leaseOwner,omitempty"`
	LeaseExpiresAt time.Time             `json:"leaseExpiresAt,omitempty"`
	Result         map[string]any        `json:"result,omitempty"`
	Error          string                `json:"error,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type CreateClusterRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	PrometheusURL string `json:"prometheusUrl,omitempty"`
	GrafanaURL    string `json:"grafanaUrl,omitempty"`
}

type RegisterAgentRequest struct {
	ClusterID    string            `json:"clusterId"`
	Version      string            `json:"version,omitempty"`
	Capabilities map[string]string `json:"capabilities,omitempty"`
}

type HeartbeatRequest struct {
	Version                 string            `json:"version,omitempty"`
	Capabilities            map[string]string `json:"capabilities,omitempty"`
	LastInventoryRevision   string            `json:"lastInventoryRevision,omitempty"`
	LastInventoryFreshness  string            `json:"lastInventoryFreshness,omitempty"`
	LastInventoryObservedAt time.Time         `json:"lastInventoryObservedAt,omitempty"`
}

type CreateAcceleratorPoolRequest struct {
	ClusterID    string            `json:"clusterId"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
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
	ClusterID string                `json:"clusterId"`
	Type      platformtask.TaskType `json:"type"`
	Payload   map[string]any        `json:"payload,omitempty"`
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

type CreateDiagnosticsTaskRequest struct {
	ServingApplicationID string `json:"servingApplicationId"`
}

type LeaseTaskRequest struct {
	AgentID string `json:"agentId"`
}

type RenewTaskLeaseRequest struct {
	AgentID string `json:"agentId"`
}

type CompleteTaskRequest struct {
	AgentID string         `json:"agentId"`
	Status  TaskStatus     `json:"status"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}
