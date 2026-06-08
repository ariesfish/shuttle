package task

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrUnsupportedType = errors.New("unsupported task type")
	ErrInvalidPayload  = errors.New("invalid task payload")
	ErrInvalidResult   = errors.New("invalid task result")
)

type Type string

const (
	TypePreviewDeploymentDiff Type = "PreviewDeploymentDiff"
	TypeApplyDeployment       Type = "ApplyDeployment"
	TypeDeleteBeforeApply     Type = "DeleteBeforeApplyRedeploy"
	TypeRetireDeployment      Type = "RetireDeployment"
	TypeFetchDiagnostics      Type = "FetchDiagnostics"
)

type DTO struct {
	ID        string
	ClusterID string
	Type      Type
	Payload   map[string]any
	Result    map[string]any
	Error     string
}

type Envelope struct {
	ClusterID string
	Type      Type
	Payload   Payload
}

type ResourceRef struct {
	Name      string
	Namespace string
}

type Manifest struct {
	Name    string
	Content string
}

type EndpointIntent struct {
	Name     string
	Protocol string
	Exposure string
}

type RenderedDeploymentTaskInput struct {
	Type                 Type
	ServingApplicationID string
	ClusterID            string
	Resource             ResourceRef
	Endpoint             EndpointIntent
	Manifests            []Manifest
}

type ResourceTaskInput struct {
	Type                 Type
	ServingApplicationID string
	ClusterID            string
	Resource             ResourceRef
}

type Payload interface {
	TaskType() Type
	ServingApplicationID() string
	Resource() ResourceRef
}

type RenderedDeploymentPayload struct {
	TypeValue                 Type
	ServingApplicationIDValue string
	ResourceValue             ResourceRef
	EndpointValue             EndpointIntent
	ManifestValues            []Manifest
}

func (p RenderedDeploymentPayload) TaskType() Type               { return p.TypeValue }
func (p RenderedDeploymentPayload) ServingApplicationID() string { return p.ServingApplicationIDValue }
func (p RenderedDeploymentPayload) Resource() ResourceRef        { return p.ResourceValue }
func (p RenderedDeploymentPayload) Endpoint() EndpointIntent     { return p.EndpointValue }
func (p RenderedDeploymentPayload) Manifests() []Manifest {
	return append([]Manifest(nil), p.ManifestValues...)
}

type ResourcePayload struct {
	TypeValue                 Type
	ServingApplicationIDValue string
	ResourceValue             ResourceRef
}

func (p ResourcePayload) TaskType() Type               { return p.TypeValue }
func (p ResourcePayload) ServingApplicationID() string { return p.ServingApplicationIDValue }
func (p ResourcePayload) Resource() ResourceRef        { return p.ResourceValue }

type Result interface {
	TaskType() Type
}

type PreviewResult struct {
	ManifestCount int
	Stdout        string
	Stderr        string
	HandledAt     time.Time
}

func (r PreviewResult) TaskType() Type { return TypePreviewDeploymentDiff }

type DeploymentResult struct {
	TypeValue          Type
	ManifestCount      int
	Stdout             string
	Stderr             string
	Resource           ResourceRef
	EndpointURL        string
	Phase              string
	Message            string
	DeletedBeforeApply bool
	DeleteMessage      string
	HandledAt          time.Time
}

func (r DeploymentResult) TaskType() Type { return r.TypeValue }

type RetireResult struct {
	Resource  ResourceRef
	Deleted   bool
	Message   string
	HandledAt time.Time
}

func (r RetireResult) TaskType() Type { return TypeRetireDeployment }

type DiagnosticsSection struct {
	Name   string
	Output string
	Error  string
}

type DiagnosticsResult struct {
	Resource  ResourceRef
	Sections  []DiagnosticsSection
	HandledAt time.Time
}

func (r DiagnosticsResult) TaskType() Type { return TypeFetchDiagnostics }

func BuildRenderedDeploymentTask(input RenderedDeploymentTaskInput) (Envelope, error) {
	if input.Type != TypePreviewDeploymentDiff && input.Type != TypeApplyDeployment && input.Type != TypeDeleteBeforeApply {
		return Envelope{}, fmt.Errorf("%w: %s", ErrUnsupportedType, input.Type)
	}
	if strings.TrimSpace(input.ServingApplicationID) == "" || strings.TrimSpace(input.ClusterID) == "" {
		return Envelope{}, fmt.Errorf("%w: servingApplicationId and clusterId are required", ErrInvalidPayload)
	}
	resource := normalizeResource(input.Resource)
	if err := validateResource(resource); err != nil {
		return Envelope{}, err
	}
	manifests := normalizeManifests(input.Manifests)
	if len(manifests) == 0 {
		return Envelope{}, fmt.Errorf("%w: at least one manifest is required", ErrInvalidPayload)
	}
	payload := RenderedDeploymentPayload{
		TypeValue:                 input.Type,
		ServingApplicationIDValue: strings.TrimSpace(input.ServingApplicationID),
		ResourceValue:             resource,
		EndpointValue:             EndpointIntent{Name: strings.TrimSpace(input.Endpoint.Name), Protocol: strings.TrimSpace(input.Endpoint.Protocol), Exposure: strings.TrimSpace(input.Endpoint.Exposure)},
		ManifestValues:            manifests,
	}
	return Envelope{ClusterID: strings.TrimSpace(input.ClusterID), Type: input.Type, Payload: payload}, nil
}

func BuildResourceTask(input ResourceTaskInput) (Envelope, error) {
	if input.Type != TypeRetireDeployment && input.Type != TypeFetchDiagnostics {
		return Envelope{}, fmt.Errorf("%w: %s", ErrUnsupportedType, input.Type)
	}
	if strings.TrimSpace(input.ServingApplicationID) == "" || strings.TrimSpace(input.ClusterID) == "" {
		return Envelope{}, fmt.Errorf("%w: servingApplicationId and clusterId are required", ErrInvalidPayload)
	}
	resource := normalizeResource(input.Resource)
	if err := validateResource(resource); err != nil {
		return Envelope{}, err
	}
	payload := ResourcePayload{TypeValue: input.Type, ServingApplicationIDValue: strings.TrimSpace(input.ServingApplicationID), ResourceValue: resource}
	return Envelope{ClusterID: strings.TrimSpace(input.ClusterID), Type: input.Type, Payload: payload}, nil
}

func DecodePayload(dto DTO) (Payload, error) {
	switch dto.Type {
	case TypePreviewDeploymentDiff, TypeApplyDeployment, TypeDeleteBeforeApply:
		appID, _ := dto.Payload["servingApplicationId"].(string)
		if strings.TrimSpace(appID) == "" {
			return nil, fmt.Errorf("%w: servingApplicationId is required", ErrInvalidPayload)
		}
		resource, err := resourceFromMap(dto.Payload)
		if err != nil {
			return nil, err
		}
		manifests, err := manifestsFromMap(dto.Payload)
		if err != nil {
			return nil, err
		}
		payload := RenderedDeploymentPayload{
			TypeValue:                 dto.Type,
			ServingApplicationIDValue: strings.TrimSpace(appID),
			ResourceValue:             resource,
			EndpointValue: EndpointIntent{
				Name:     stringField(dto.Payload, "endpointName"),
				Protocol: stringField(dto.Payload, "protocol"),
				Exposure: stringField(dto.Payload, "exposure"),
			},
			ManifestValues: manifests,
		}
		return payload, nil
	case TypeRetireDeployment, TypeFetchDiagnostics:
		appID, _ := dto.Payload["servingApplicationId"].(string)
		if strings.TrimSpace(appID) == "" {
			return nil, fmt.Errorf("%w: servingApplicationId is required", ErrInvalidPayload)
		}
		resource, err := resourceFromMap(dto.Payload)
		if err != nil {
			return nil, err
		}
		payload := ResourcePayload{TypeValue: dto.Type, ServingApplicationIDValue: strings.TrimSpace(appID), ResourceValue: resource}
		return payload, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dto.Type)
	}
}

func DecodeResult(dto DTO) (Result, error) {
	switch dto.Type {
	case TypePreviewDeploymentDiff:
		return PreviewResult{ManifestCount: intField(dto.Result, "manifestCount"), Stdout: stringField(dto.Result, "stdout"), Stderr: stringField(dto.Result, "stderr"), HandledAt: parseHandledAt(dto.Result)}, nil
	case TypeApplyDeployment, TypeDeleteBeforeApply:
		resource, err := resourceFromMap(dto.Result)
		if err != nil {
			resource, err = resourceFromMap(dto.Payload)
			if err != nil {
				return nil, err
			}
		}
		return DeploymentResult{TypeValue: dto.Type, ManifestCount: intField(dto.Result, "manifestCount"), Stdout: stringField(dto.Result, "stdout"), Stderr: stringField(dto.Result, "stderr"), Resource: resource, EndpointURL: stringField(dto.Result, "endpointUrl"), Phase: stringField(dto.Result, "phase"), Message: stringField(dto.Result, "message"), DeletedBeforeApply: boolField(dto.Result, "deletedBeforeApply"), DeleteMessage: stringField(dto.Result, "deleteMessage"), HandledAt: parseHandledAt(dto.Result)}, nil
	case TypeRetireDeployment:
		resource, err := resourceFromMap(dto.Result)
		if err != nil {
			resource, err = resourceFromMap(dto.Payload)
			if err != nil {
				return nil, err
			}
		}
		return RetireResult{Resource: resource, Deleted: boolField(dto.Result, "deleted"), Message: stringField(dto.Result, "message"), HandledAt: parseHandledAt(dto.Result)}, nil
	case TypeFetchDiagnostics:
		resource, err := resourceFromMap(dto.Result)
		if err != nil {
			resource, err = resourceFromMap(dto.Payload)
			if err != nil {
				return nil, err
			}
		}
		return DiagnosticsResult{Resource: resource, Sections: sectionsFromMap(dto.Result), HandledAt: parseHandledAt(dto.Result)}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dto.Type)
	}
}

func EncodePayload(payload Payload) map[string]any {
	base := map[string]any{
		"servingApplicationId": payload.ServingApplicationID(),
		"resourceName":         payload.Resource().Name,
		"namespace":            payload.Resource().Namespace,
	}
	if rendered, ok := payload.(RenderedDeploymentPayload); ok {
		endpoint := rendered.Endpoint()
		base["endpointName"] = endpoint.Name
		base["protocol"] = endpoint.Protocol
		base["exposure"] = endpoint.Exposure
		manifests := make([]any, 0, len(rendered.ManifestValues))
		for _, manifest := range rendered.ManifestValues {
			manifests = append(manifests, map[string]any{"name": manifest.Name, "content": manifest.Content})
		}
		base["manifests"] = manifests
	}
	return base
}

func EncodeResult(result Result) map[string]any {
	switch value := result.(type) {
	case PreviewResult:
		return withHandledAt(map[string]any{"mode": "server-side-dry-run", "manifestCount": value.ManifestCount, "stdout": value.Stdout, "stderr": value.Stderr}, value.HandledAt)
	case DeploymentResult:
		output := map[string]any{"mode": deploymentMode(value.TypeValue), "manifestCount": value.ManifestCount, "stdout": value.Stdout, "stderr": value.Stderr, "resource": value.Resource.Name, "namespace": value.Resource.Namespace, "endpointUrl": value.EndpointURL, "phase": value.Phase, "message": value.Message}
		if value.TypeValue == TypeDeleteBeforeApply {
			output["deletedBeforeApply"] = value.DeletedBeforeApply
			output["deleteMessage"] = value.DeleteMessage
		}
		return withHandledAt(output, value.HandledAt)
	case RetireResult:
		return withHandledAt(map[string]any{"mode": "retire", "resource": value.Resource.Name, "namespace": value.Resource.Namespace, "deleted": value.Deleted, "message": value.Message}, value.HandledAt)
	case DiagnosticsResult:
		sections := make([]any, 0, len(value.Sections))
		for _, section := range value.Sections {
			sections = append(sections, map[string]any{"name": section.Name, "output": section.Output, "error": section.Error})
		}
		return withHandledAt(map[string]any{"mode": "diagnostics", "resource": value.Resource.Name, "namespace": value.Resource.Namespace, "sections": sections}, value.HandledAt)
	default:
		return map[string]any{}
	}
}

func normalizeResource(ref ResourceRef) ResourceRef {
	return ResourceRef{Name: strings.TrimSpace(ref.Name), Namespace: strings.TrimSpace(ref.Namespace)}
}

func validateResource(ref ResourceRef) error {
	if ref.Name == "" || ref.Namespace == "" {
		return fmt.Errorf("%w: resource name and namespace are required", ErrInvalidPayload)
	}
	return nil
}

func normalizeManifests(input []Manifest) []Manifest {
	manifests := make([]Manifest, 0, len(input))
	for _, manifest := range input {
		name := strings.TrimSpace(manifest.Name)
		content := strings.TrimSpace(manifest.Content)
		if content == "" {
			continue
		}
		manifests = append(manifests, Manifest{Name: name, Content: manifest.Content})
	}
	return manifests
}

func resourceFromMap(values map[string]any) (ResourceRef, error) {
	name := stringField(values, "resourceName")
	if name == "" {
		name = stringField(values, "resource")
	}
	ref := ResourceRef{Name: name, Namespace: stringField(values, "namespace")}
	if err := validateResource(ref); err != nil {
		return ResourceRef{}, err
	}
	return ref, nil
}

func manifestsFromMap(values map[string]any) ([]Manifest, error) {
	rawManifests, ok := values["manifests"]
	if !ok {
		return nil, fmt.Errorf("%w: manifests is required", ErrInvalidPayload)
	}
	items, ok := rawManifests.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: manifests must be an array", ErrInvalidPayload)
	}
	manifests := make([]Manifest, 0, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: manifests[%d] must be an object", ErrInvalidPayload, index)
		}
		name := stringField(object, "name")
		content := stringField(object, "content")
		if content == "" {
			return nil, fmt.Errorf("%w: manifests[%d].content is required", ErrInvalidPayload, index)
		}
		manifests = append(manifests, Manifest{Name: name, Content: content})
	}
	if len(manifests) == 0 {
		return nil, fmt.Errorf("%w: at least one manifest is required", ErrInvalidPayload)
	}
	return manifests, nil
}

func sectionsFromMap(values map[string]any) []DiagnosticsSection {
	rawSections, ok := values["sections"].([]any)
	if !ok {
		return nil
	}
	sections := make([]DiagnosticsSection, 0, len(rawSections))
	for _, raw := range rawSections {
		object, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		sections = append(sections, DiagnosticsSection{Name: stringField(object, "name"), Output: stringField(object, "output"), Error: stringField(object, "error")})
	}
	return sections
}

func stringField(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func intField(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func boolField(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}

func parseHandledAt(values map[string]any) time.Time {
	value := stringField(values, "handledAt")
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func withHandledAt(values map[string]any, handledAt time.Time) map[string]any {
	if handledAt.IsZero() {
		handledAt = time.Now().UTC()
	}
	values["handledAt"] = handledAt.UTC().Format(time.RFC3339)
	return values
}

func deploymentMode(taskType Type) string {
	if taskType == TypeDeleteBeforeApply {
		return "delete-before-apply"
	}
	return "apply-and-watch"
}
