package task

// PayloadKind describes the shape a task payload must have at the Cluster Agent seam.
type PayloadKind string

const (
	PayloadKindNone               PayloadKind = "none"
	PayloadKindRenderedDeployment PayloadKind = "rendered-deployment"
	PayloadKindResource           PayloadKind = "resource"
)

// ResultKind describes the shape a task result produces at the Cluster Agent seam.
type ResultKind string

const (
	ResultKindNone              ResultKind = "none"
	ResultKindPreview           ResultKind = "preview"
	ResultKindDeployment        ResultKind = "deployment"
	ResultKindRetire            ResultKind = "retire"
	ResultKindDiagnostics       ResultKind = "diagnostics"
	ResultKindEndpointReadiness ResultKind = "endpoint-readiness"
)

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

// Definition is the authoritative Cluster Agent task contract shared by the
// Management Plane and Cluster Agent. It keeps the whitelist, payload shape,
// and result shape in one place instead of scattering switch statements across
// modules.
type Definition struct {
	Type            TaskType
	PayloadKind     PayloadKind
	ResultKind      ResultKind
	AgentExecutable bool
}

var definitions = map[TaskType]Definition{
	TaskTypeRegisterCluster:       {Type: TaskTypeRegisterCluster, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone},
	TaskTypeValidateIntent:        {Type: TaskTypeValidateIntent, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone},
	TaskTypePreviewDeploymentDiff: {Type: TaskTypePreviewDeploymentDiff, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindPreview, AgentExecutable: true},
	TaskTypeApplyDeployment:       {Type: TaskTypeApplyDeployment, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindDeployment, AgentExecutable: true},
	TaskTypeDeleteBeforeApply:     {Type: TaskTypeDeleteBeforeApply, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindDeployment, AgentExecutable: true},
	TaskTypeInspectStatus:         {Type: TaskTypeInspectStatus, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone},
	TaskTypeRetireDeployment:      {Type: TaskTypeRetireDeployment, PayloadKind: PayloadKindResource, ResultKind: ResultKindRetire, AgentExecutable: true},
	TaskTypeFetchDiagnostics:      {Type: TaskTypeFetchDiagnostics, PayloadKind: PayloadKindResource, ResultKind: ResultKindDiagnostics, AgentExecutable: true},
	TaskTypeSyncEndpointReadiness: {Type: TaskTypeSyncEndpointReadiness, PayloadKind: PayloadKindNone, ResultKind: ResultKindEndpointReadiness},
}

func DefinitionFor(taskType TaskType) (Definition, bool) {
	definition, ok := definitions[taskType]
	return definition, ok
}

func IsWhitelisted(taskType TaskType) bool {
	_, ok := definitions[taskType]
	return ok
}

func IsAgentExecutable(taskType TaskType) bool {
	definition, ok := definitions[taskType]
	return ok && definition.AgentExecutable
}

func PayloadKindFor(taskType TaskType) PayloadKind {
	definition, ok := definitions[taskType]
	if !ok {
		return PayloadKindNone
	}
	return definition.PayloadKind
}

func NewDTO(id string, clusterID string, taskType TaskType, payload map[string]any, result map[string]any, taskError string) DTO {
	return DTO{ID: id, ClusterID: clusterID, Type: taskType, Payload: payload, Result: result, Error: taskError}
}
