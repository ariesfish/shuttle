package task

import (
	"errors"
	"testing"
)

func TestRegistryDefinitionsDescribeTaskContract(t *testing.T) {
	registry := DefaultRegistry()
	cases := []struct {
		taskType        TaskType
		payloadKind     PayloadKind
		resultKind      ResultKind
		agentExecutable bool
		effect          LifecycleEffect
	}{
		{TaskTypePreviewDeploymentDiff, PayloadKindRenderedDeployment, ResultKindPreview, true, LifecycleEffectPreview},
		{TaskTypeApplyDeployment, PayloadKindRenderedDeployment, ResultKindDeployment, true, LifecycleEffectDeployment},
		{TaskTypeDeleteBeforeApply, PayloadKindRenderedDeployment, ResultKindDeployment, true, LifecycleEffectDeployment},
		{TaskTypeRetireDeployment, PayloadKindResource, ResultKindRetire, true, LifecycleEffectRetire},
		{TaskTypeFetchDiagnostics, PayloadKindResource, ResultKindDiagnostics, true, LifecycleEffectDiagnostics},
		{TaskTypeInspectStatus, PayloadKindNone, ResultKindNone, false, LifecycleEffectNone},
	}
	for _, tc := range cases {
		if registry.PayloadKindFor(tc.taskType) != tc.payloadKind || registry.ResultKindFor(tc.taskType) != tc.resultKind || registry.IsAgentExecutable(tc.taskType) != tc.agentExecutable || registry.LifecycleEffectFor(tc.taskType) != tc.effect {
			t.Fatalf("unexpected definition for %s: %+v", tc.taskType, registry.definitions[tc.taskType])
		}
	}
	if registry.IsWhitelisted(TaskType("ArbitraryKubectl")) || registry.LifecycleEffectFor(TaskType("ArbitraryKubectl")) != LifecycleEffectNone {
		t.Fatalf("unsupported task type should not be whitelisted")
	}
}

func TestBuildRenderedDeploymentTaskEncodesPayload(t *testing.T) {
	envelope, err := BuildRenderedDeploymentTask(RenderedDeploymentTaskInput{
		Type:                 TaskTypeApplyDeployment,
		ServingApplicationID: "app-1",
		ClusterID:            "cluster-1",
		Resource:             ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
		Endpoint:             EndpointIntent{Name: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
		Manifests:            []Manifest{{Name: "dgd.yaml", Content: "kind: DynamoGraphDeployment\n"}},
	})
	if err != nil {
		t.Fatalf("build rendered deployment task: %v", err)
	}
	if envelope.ClusterID != "cluster-1" || envelope.Type != TaskTypeApplyDeployment {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	payload := EncodePayload(envelope.Payload)
	if payload["servingApplicationId"] != "app-1" || payload["resourceName"] != "deepseek-v4-flash" || payload["namespace"] != "dynamo-system" {
		t.Fatalf("unexpected payload identity fields: %+v", payload)
	}
	manifests, ok := payload["manifests"].([]any)
	if !ok || len(manifests) != 1 {
		t.Fatalf("expected encoded manifests, got %+v", payload["manifests"])
	}
}

func TestBuildRenderedDeploymentTaskRejectsUnsafeShape(t *testing.T) {
	_, err := BuildRenderedDeploymentTask(RenderedDeploymentTaskInput{
		Type:                 TaskTypeApplyDeployment,
		ServingApplicationID: "app-1",
		ClusterID:            "cluster-1",
		Resource:             ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
	})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}

	_, err = BuildRenderedDeploymentTask(RenderedDeploymentTaskInput{
		Type:                 TaskType("ArbitraryKubectl"),
		ServingApplicationID: "app-1",
		ClusterID:            "cluster-1",
		Resource:             ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"},
		Manifests:            []Manifest{{Content: "kind: ConfigMap\n"}},
	})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
}

func TestDecodePayloadRequiresServingApplicationIdentity(t *testing.T) {
	_, err := DecodePayload(DTO{
		Type: TaskTypePreviewDeploymentDiff,
		Payload: map[string]any{
			"resourceName": "deepseek-v4-flash",
			"namespace":    "dynamo-system",
			"manifests":    []any{map[string]any{"name": "preview.yaml", "content": "kind: ConfigMap\n"}},
		},
	})
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}

func TestDecodeResultFallsBackToPayloadResource(t *testing.T) {
	result, err := DecodeResult(DTO{
		Type: TaskTypeApplyDeployment,
		Payload: map[string]any{
			"servingApplicationId": "app-1",
			"resourceName":         "deepseek-v4-flash",
			"namespace":            "dynamo-system",
		},
		Result: map[string]any{
			"phase":       "Ready",
			"endpointUrl": "http://deepseek-v4-flash.dynamo-system.svc.cluster.local:8000/v1",
		},
	})
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	deployment, ok := result.(DeploymentResult)
	if !ok || deployment.Resource.Name != "deepseek-v4-flash" || deployment.Phase != "Ready" {
		t.Fatalf("unexpected deployment result: %+v", result)
	}
}
