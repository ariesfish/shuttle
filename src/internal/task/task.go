package task

import (
	"encoding/json"
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

type TaskType string

type DTO struct {
	ID        string
	ClusterID string
	Type      TaskType
	Payload   map[string]any
	Result    map[string]any
	Error     string
}

type encodedPayload struct {
	ServingApplicationID string            `json:"servingApplicationId,omitempty"`
	ResourceName         string            `json:"resourceName,omitempty"`
	Namespace            string            `json:"namespace,omitempty"`
	EndpointName         string            `json:"endpointName,omitempty"`
	Protocol             string            `json:"protocol,omitempty"`
	Exposure             string            `json:"exposure,omitempty"`
	Manifests            []encodedManifest `json:"manifests,omitempty"`
}

type encodedManifest struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`
}

type encodedResult struct {
	Mode               string                      `json:"mode,omitempty"`
	ManifestCount      int                         `json:"manifestCount,omitempty"`
	Stdout             string                      `json:"stdout,omitempty"`
	Stderr             string                      `json:"stderr,omitempty"`
	Resource           string                      `json:"resource,omitempty"`
	ResourceName       string                      `json:"resourceName,omitempty"`
	Namespace          string                      `json:"namespace,omitempty"`
	EndpointURL        string                      `json:"endpointUrl,omitempty"`
	Phase              string                      `json:"phase,omitempty"`
	Message            string                      `json:"message,omitempty"`
	DeletedBeforeApply bool                        `json:"deletedBeforeApply,omitempty"`
	DeleteMessage      string                      `json:"deleteMessage,omitempty"`
	Deleted            bool                        `json:"deleted,omitempty"`
	Sections           []encodedDiagnosticsSection `json:"sections,omitempty"`
	HandledAt          string                      `json:"handledAt,omitempty"`
}

type encodedDiagnosticsSection struct {
	Name   string `json:"name,omitempty"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Envelope struct {
	ClusterID string
	Type      TaskType
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
	Type                 TaskType
	ServingApplicationID string
	ClusterID            string
	Resource             ResourceRef
	Endpoint             EndpointIntent
	Manifests            []Manifest
}

type ResourceTaskInput struct {
	Type                 TaskType
	ServingApplicationID string
	ClusterID            string
	Resource             ResourceRef
}

type Payload interface {
	TaskType() TaskType
	ServingApplicationID() string
	Resource() ResourceRef
}

type RenderedDeploymentPayload struct {
	TypeValue                 TaskType
	ServingApplicationIDValue string
	ResourceValue             ResourceRef
	EndpointValue             EndpointIntent
	ManifestValues            []Manifest
}

func (p RenderedDeploymentPayload) TaskType() TaskType           { return p.TypeValue }
func (p RenderedDeploymentPayload) ServingApplicationID() string { return p.ServingApplicationIDValue }
func (p RenderedDeploymentPayload) Resource() ResourceRef        { return p.ResourceValue }
func (p RenderedDeploymentPayload) Endpoint() EndpointIntent     { return p.EndpointValue }
func (p RenderedDeploymentPayload) Manifests() []Manifest {
	return append([]Manifest(nil), p.ManifestValues...)
}

type ResourcePayload struct {
	TypeValue                 TaskType
	ServingApplicationIDValue string
	ResourceValue             ResourceRef
}

func (p ResourcePayload) TaskType() TaskType           { return p.TypeValue }
func (p ResourcePayload) ServingApplicationID() string { return p.ServingApplicationIDValue }
func (p ResourcePayload) Resource() ResourceRef        { return p.ResourceValue }

type Result interface {
	TaskType() TaskType
}

type PreviewResult struct {
	ManifestCount int
	Stdout        string
	Stderr        string
	HandledAt     time.Time
}

func (r PreviewResult) TaskType() TaskType { return TaskTypePreviewDeploymentDiff }

type DeploymentResult struct {
	TypeValue          TaskType
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

func (r DeploymentResult) TaskType() TaskType { return r.TypeValue }

type RetireResult struct {
	Resource  ResourceRef
	Deleted   bool
	Message   string
	HandledAt time.Time
}

func (r RetireResult) TaskType() TaskType { return TaskTypeRetireDeployment }

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

func (r DiagnosticsResult) TaskType() TaskType { return TaskTypeFetchDiagnostics }

func BuildRenderedDeploymentTask(input RenderedDeploymentTaskInput) (Envelope, error) {
	return DefaultRegistry().BuildRenderedDeployment(input)
}

func buildRenderedDeploymentTask(registry Registry, input RenderedDeploymentTaskInput) (Envelope, error) {
	if registry.PayloadKindFor(input.Type) != PayloadKindRenderedDeployment {
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
	return DefaultRegistry().BuildResource(input)
}

func buildResourceTask(registry Registry, input ResourceTaskInput) (Envelope, error) {
	if registry.PayloadKindFor(input.Type) != PayloadKindResource {
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
	return DefaultRegistry().DecodePayload(dto)
}

func decodePayload(registry Registry, dto DTO) (Payload, error) {
	payloadFields, err := decodePayloadFields(dto.Payload)
	if err != nil {
		return nil, err
	}
	appID := strings.TrimSpace(payloadFields.ServingApplicationID)
	if appID == "" && registry.PayloadKindFor(dto.Type) != PayloadKindNone {
		return nil, fmt.Errorf("%w: servingApplicationId is required", ErrInvalidPayload)
	}
	resource, err := resourceFromPayloadFields(payloadFields)
	if err != nil && registry.PayloadKindFor(dto.Type) != PayloadKindNone {
		return nil, err
	}

	switch registry.PayloadKindFor(dto.Type) {
	case PayloadKindRenderedDeployment:
		manifests, err := manifestsFromPayloadFields(payloadFields)
		if err != nil {
			return nil, err
		}
		payload := RenderedDeploymentPayload{
			TypeValue:                 dto.Type,
			ServingApplicationIDValue: appID,
			ResourceValue:             resource,
			EndpointValue: EndpointIntent{
				Name:     strings.TrimSpace(payloadFields.EndpointName),
				Protocol: strings.TrimSpace(payloadFields.Protocol),
				Exposure: strings.TrimSpace(payloadFields.Exposure),
			},
			ManifestValues: manifests,
		}
		return payload, nil
	case PayloadKindResource:
		payload := ResourcePayload{TypeValue: dto.Type, ServingApplicationIDValue: appID, ResourceValue: resource}
		return payload, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dto.Type)
	}
}

func DecodeResult(dto DTO) (Result, error) {
	return DefaultRegistry().DecodeResult(dto)
}

func decodeResult(registry Registry, dto DTO) (Result, error) {
	resultFields, err := decodeResultFields(dto.Result)
	if err != nil {
		return nil, err
	}
	switch registry.PayloadKindFor(dto.Type) {
	case PayloadKindRenderedDeployment:
		if dto.Type == TaskTypePreviewDeploymentDiff {
			return PreviewResult{ManifestCount: resultFields.ManifestCount, Stdout: strings.TrimSpace(resultFields.Stdout), Stderr: strings.TrimSpace(resultFields.Stderr), HandledAt: parseHandledAtValue(resultFields.HandledAt)}, nil
		}
		resource, err := resourceFromResultOrPayload(resultFields, dto.Payload)
		if err != nil {
			return nil, err
		}
		return DeploymentResult{TypeValue: dto.Type, ManifestCount: resultFields.ManifestCount, Stdout: strings.TrimSpace(resultFields.Stdout), Stderr: strings.TrimSpace(resultFields.Stderr), Resource: resource, EndpointURL: strings.TrimSpace(resultFields.EndpointURL), Phase: strings.TrimSpace(resultFields.Phase), Message: strings.TrimSpace(resultFields.Message), DeletedBeforeApply: resultFields.DeletedBeforeApply, DeleteMessage: strings.TrimSpace(resultFields.DeleteMessage), HandledAt: parseHandledAtValue(resultFields.HandledAt)}, nil
	case PayloadKindResource:
		resource, err := resourceFromResultOrPayload(resultFields, dto.Payload)
		if err != nil {
			return nil, err
		}
		if dto.Type == TaskTypeFetchDiagnostics {
			return DiagnosticsResult{Resource: resource, Sections: diagnosticsSectionsFromFields(resultFields), HandledAt: parseHandledAtValue(resultFields.HandledAt)}, nil
		}
		return RetireResult{Resource: resource, Deleted: resultFields.Deleted, Message: strings.TrimSpace(resultFields.Message), HandledAt: parseHandledAtValue(resultFields.HandledAt)}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, dto.Type)
	}
}

func EncodePayload(payload Payload) map[string]any {
	fields := encodedPayload{
		ServingApplicationID: payload.ServingApplicationID(),
		ResourceName:         payload.Resource().Name,
		Namespace:            payload.Resource().Namespace,
	}
	if rendered, ok := payload.(RenderedDeploymentPayload); ok {
		endpoint := rendered.Endpoint()
		fields.EndpointName = endpoint.Name
		fields.Protocol = endpoint.Protocol
		fields.Exposure = endpoint.Exposure
		fields.Manifests = make([]encodedManifest, 0, len(rendered.ManifestValues))
		for _, manifest := range rendered.ManifestValues {
			fields.Manifests = append(fields.Manifests, encodedManifest{Name: manifest.Name, Content: manifest.Content})
		}
	}
	return encodeFields(fields)
}

func EncodeResult(result Result) map[string]any {
	switch value := result.(type) {
	case PreviewResult:
		return encodeResultFields(encodedResult{Mode: "server-side-dry-run", ManifestCount: value.ManifestCount, Stdout: value.Stdout, Stderr: value.Stderr}, value.HandledAt)
	case DeploymentResult:
		fields := encodedResult{Mode: deploymentMode(value.TypeValue), ManifestCount: value.ManifestCount, Stdout: value.Stdout, Stderr: value.Stderr, Resource: value.Resource.Name, Namespace: value.Resource.Namespace, EndpointURL: value.EndpointURL, Phase: value.Phase, Message: value.Message}
		if value.TypeValue == TaskTypeDeleteBeforeApply {
			fields.DeletedBeforeApply = value.DeletedBeforeApply
			fields.DeleteMessage = value.DeleteMessage
		}
		return encodeResultFields(fields, value.HandledAt)
	case RetireResult:
		return encodeResultFields(encodedResult{Mode: "retire", Resource: value.Resource.Name, Namespace: value.Resource.Namespace, Deleted: value.Deleted, Message: value.Message}, value.HandledAt)
	case DiagnosticsResult:
		sections := make([]encodedDiagnosticsSection, 0, len(value.Sections))
		for _, section := range value.Sections {
			sections = append(sections, encodedDiagnosticsSection{Name: section.Name, Output: section.Output, Error: section.Error})
		}
		return encodeResultFields(encodedResult{Mode: "diagnostics", Resource: value.Resource.Name, Namespace: value.Resource.Namespace, Sections: sections}, value.HandledAt)
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

func decodePayloadFields(values map[string]any) (encodedPayload, error) {
	var fields encodedPayload
	if err := decodeFields(values, &fields); err != nil {
		return encodedPayload{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return fields, nil
}

func decodeResultFields(values map[string]any) (encodedResult, error) {
	var fields encodedResult
	if err := decodeFields(values, &fields); err != nil {
		return encodedResult{}, fmt.Errorf("%w: %v", ErrInvalidResult, err)
	}
	return fields, nil
}

func decodeFields[T any](values map[string]any, target *T) error {
	if values == nil {
		values = map[string]any{}
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

func encodeFields[T any](fields T) map[string]any {
	bytes, err := json.Marshal(fields)
	if err != nil {
		return map[string]any{}
	}
	decoder := json.NewDecoder(strings.NewReader(string(bytes)))
	decoder.UseNumber()
	var output map[string]any
	if err := decoder.Decode(&output); err != nil {
		return map[string]any{}
	}
	return normalizeEncodedMap(output)
}

func normalizeEncodedMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = normalizeEncodedValue(value)
	}
	return output
}

func normalizeEncodedValue(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return int(intValue)
		}
		floatValue, _ := typed.Float64()
		return floatValue
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, normalizeEncodedValue(item))
		}
		return items
	case map[string]any:
		return normalizeEncodedMap(typed)
	default:
		return value
	}
}

func encodeResultFields(fields encodedResult, handledAt time.Time) map[string]any {
	if handledAt.IsZero() {
		handledAt = time.Now().UTC()
	}
	fields.HandledAt = handledAt.UTC().Format(time.RFC3339)
	return encodeFields(fields)
}

func resourceFromPayloadFields(fields encodedPayload) (ResourceRef, error) {
	ref := ResourceRef{Name: strings.TrimSpace(fields.ResourceName), Namespace: strings.TrimSpace(fields.Namespace)}
	if err := validateResource(ref); err != nil {
		return ResourceRef{}, err
	}
	return ref, nil
}

func resourceFromResultOrPayload(result encodedResult, payload map[string]any) (ResourceRef, error) {
	name := strings.TrimSpace(result.ResourceName)
	if name == "" {
		name = strings.TrimSpace(result.Resource)
	}
	ref := ResourceRef{Name: name, Namespace: strings.TrimSpace(result.Namespace)}
	if err := validateResource(ref); err == nil {
		return ref, nil
	}
	payloadFields, err := decodePayloadFields(payload)
	if err != nil {
		return ResourceRef{}, err
	}
	return resourceFromPayloadFields(payloadFields)
}

func manifestsFromPayloadFields(fields encodedPayload) ([]Manifest, error) {
	if len(fields.Manifests) == 0 {
		return nil, fmt.Errorf("%w: at least one manifest is required", ErrInvalidPayload)
	}
	manifests := make([]Manifest, 0, len(fields.Manifests))
	for index, item := range fields.Manifests {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			return nil, fmt.Errorf("%w: manifests[%d].content is required", ErrInvalidPayload, index)
		}
		manifests = append(manifests, Manifest{Name: strings.TrimSpace(item.Name), Content: item.Content})
	}
	return manifests, nil
}

func diagnosticsSectionsFromFields(fields encodedResult) []DiagnosticsSection {
	sections := make([]DiagnosticsSection, 0, len(fields.Sections))
	for _, section := range fields.Sections {
		sections = append(sections, DiagnosticsSection{Name: strings.TrimSpace(section.Name), Output: strings.TrimSpace(section.Output), Error: strings.TrimSpace(section.Error)})
	}
	return sections
}

func parseHandledAtValue(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func deploymentMode(taskType TaskType) string {
	if taskType == TaskTypeDeleteBeforeApply {
		return "delete-before-apply"
	}
	return "apply-and-watch"
}
