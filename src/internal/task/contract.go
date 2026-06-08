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

// LifecycleEffect describes how a completed Cluster Agent task affects a
// Serving Application in the Management Plane. It keeps task semantics at the
// task contract seam instead of scattering task-type switches across callers.
type LifecycleEffect string

const (
	LifecycleEffectNone        LifecycleEffect = "none"
	LifecycleEffectPreview     LifecycleEffect = "preview"
	LifecycleEffectDeployment  LifecycleEffect = "deployment"
	LifecycleEffectRetire      LifecycleEffect = "retire"
	LifecycleEffectDiagnostics LifecycleEffect = "diagnostics"
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
// result shape, executability, and Serving Application lifecycle effect in one
// place instead of scattering switch statements across modules.
type Definition struct {
	Type            TaskType
	PayloadKind     PayloadKind
	ResultKind      ResultKind
	AgentExecutable bool
	LifecycleEffect LifecycleEffect
}

// Registry is the deep Module for the Cluster Agent task contract. Callers use
// it to build, decode, and classify task envelopes without knowing the per-type
// payload/result rules.
type Registry struct {
	definitions map[TaskType]Definition
}

var defaultRegistry = Registry{definitions: map[TaskType]Definition{
	TaskTypeRegisterCluster:       {Type: TaskTypeRegisterCluster, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone, LifecycleEffect: LifecycleEffectNone},
	TaskTypeValidateIntent:        {Type: TaskTypeValidateIntent, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone, LifecycleEffect: LifecycleEffectNone},
	TaskTypePreviewDeploymentDiff: {Type: TaskTypePreviewDeploymentDiff, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindPreview, AgentExecutable: true, LifecycleEffect: LifecycleEffectPreview},
	TaskTypeApplyDeployment:       {Type: TaskTypeApplyDeployment, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindDeployment, AgentExecutable: true, LifecycleEffect: LifecycleEffectDeployment},
	TaskTypeDeleteBeforeApply:     {Type: TaskTypeDeleteBeforeApply, PayloadKind: PayloadKindRenderedDeployment, ResultKind: ResultKindDeployment, AgentExecutable: true, LifecycleEffect: LifecycleEffectDeployment},
	TaskTypeInspectStatus:         {Type: TaskTypeInspectStatus, PayloadKind: PayloadKindNone, ResultKind: ResultKindNone, LifecycleEffect: LifecycleEffectNone},
	TaskTypeRetireDeployment:      {Type: TaskTypeRetireDeployment, PayloadKind: PayloadKindResource, ResultKind: ResultKindRetire, AgentExecutable: true, LifecycleEffect: LifecycleEffectRetire},
	TaskTypeFetchDiagnostics:      {Type: TaskTypeFetchDiagnostics, PayloadKind: PayloadKindResource, ResultKind: ResultKindDiagnostics, AgentExecutable: true, LifecycleEffect: LifecycleEffectDiagnostics},
	TaskTypeSyncEndpointReadiness: {Type: TaskTypeSyncEndpointReadiness, PayloadKind: PayloadKindNone, ResultKind: ResultKindEndpointReadiness, LifecycleEffect: LifecycleEffectNone},
}}

func DefaultRegistry() Registry {
	return defaultRegistry
}

func (r Registry) DefinitionFor(taskType TaskType) (Definition, bool) {
	definition, ok := r.definitions[taskType]
	return definition, ok
}

func (r Registry) IsWhitelisted(taskType TaskType) bool {
	_, ok := r.DefinitionFor(taskType)
	return ok
}

func (r Registry) IsAgentExecutable(taskType TaskType) bool {
	definition, ok := r.DefinitionFor(taskType)
	return ok && definition.AgentExecutable
}

func (r Registry) PayloadKindFor(taskType TaskType) PayloadKind {
	definition, ok := r.DefinitionFor(taskType)
	if !ok {
		return PayloadKindNone
	}
	return definition.PayloadKind
}

func (r Registry) ResultKindFor(taskType TaskType) ResultKind {
	definition, ok := r.DefinitionFor(taskType)
	if !ok {
		return ResultKindNone
	}
	return definition.ResultKind
}

func (r Registry) LifecycleEffectFor(taskType TaskType) LifecycleEffect {
	definition, ok := r.DefinitionFor(taskType)
	if !ok {
		return LifecycleEffectNone
	}
	return definition.LifecycleEffect
}

func (r Registry) BuildRenderedDeployment(input RenderedDeploymentTaskInput) (Envelope, error) {
	return buildRenderedDeploymentTask(r, input)
}

func (r Registry) BuildResource(input ResourceTaskInput) (Envelope, error) {
	return buildResourceTask(r, input)
}

func (r Registry) DecodePayload(dto DTO) (Payload, error) {
	return decodePayload(r, dto)
}

func (r Registry) DecodeResult(dto DTO) (Result, error) {
	return decodeResult(r, dto)
}

func DefinitionFor(taskType TaskType) (Definition, bool) {
	return DefaultRegistry().DefinitionFor(taskType)
}

func IsWhitelisted(taskType TaskType) bool {
	return DefaultRegistry().IsWhitelisted(taskType)
}

func IsAgentExecutable(taskType TaskType) bool {
	return DefaultRegistry().IsAgentExecutable(taskType)
}

func PayloadKindFor(taskType TaskType) PayloadKind {
	return DefaultRegistry().PayloadKindFor(taskType)
}

func ResultKindFor(taskType TaskType) ResultKind {
	return DefaultRegistry().ResultKindFor(taskType)
}

func LifecycleEffectFor(taskType TaskType) LifecycleEffect {
	return DefaultRegistry().LifecycleEffectFor(taskType)
}

func NewDTO(id string, clusterID string, taskType TaskType, payload map[string]any, result map[string]any, taskError string) DTO {
	return DTO{ID: id, ClusterID: clusterID, Type: taskType, Payload: payload, Result: result, Error: taskError}
}
