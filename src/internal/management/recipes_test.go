package management

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRecipeRegistry(t *testing.T) {
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	recipes := registry.List()
	if len(recipes) != 2 {
		t.Fatalf("expected built-in recipes, got %+v", recipes)
	}
	vllmRecipe, ok := registry.Get("deepseek-v4-flash-vllm-dgd-disagg")
	if !ok || vllmRecipe.Spec.Support.Status != RecipeSupportStatusExperimental || vllmRecipe.Source != "builtin" {
		t.Fatalf("unexpected vLLM recipe: %+v", vllmRecipe)
	}
	sglangRecipe, ok := registry.Get("deepseek-v4-flash-sglang-dgd-disagg")
	if !ok || sglangRecipe.Spec.Support.Status != RecipeSupportStatusSupported || sglangRecipe.Source != "builtin" {
		t.Fatalf("unexpected SGLang recipe: %+v", sglangRecipe)
	}
}

func TestRecipeRegistryCreationPlansApplyDefaultsAndSupport(t *testing.T) {
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	artifact := ModelArtifact{ID: "artifact-1", Family: "deepseek-v4", Variant: "flash", Quantization: "fp8"}

	plans := registry.CreationPlans(artifact)
	if len(plans) != 2 {
		t.Fatalf("expected matching plans, got %+v", plans)
	}
	vllm := findCreationPlan(plans, "deepseek-v4-flash-vllm-dgd-disagg")
	if vllm == nil || vllm.Model.ArtifactID != artifact.ID || vllm.Runtime.Backend != "vllm" || vllm.Defaults.Namespace != "dynamo-system" || vllm.Defaults.Protocol != "openai-compatible" || !vllm.Creatable || vllm.Message == "" {
		t.Fatalf("unexpected vllm plan: %+v", vllm)
	}
	missing := registry.CreationPlans(ModelArtifact{ID: "artifact-2", Family: "deepseek-v4", Variant: "flash", Quantization: "int4"})
	if len(missing) != 0 {
		t.Fatalf("expected no plans for unsupported artifact, got %+v", missing)
	}
}

func TestRecipeRegistryValidateIntent(t *testing.T) {
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	artifact := ModelArtifact{Family: "deepseek-v4", Variant: "flash", Quantization: "fp8"}
	request := CreateServingApplicationRequest{
		Model:   ModelIntent{Family: "deepseek-v4", Variant: "flash", ArtifactID: "artifact-1", Quantization: "fp8"},
		Runtime: RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-vllm-dgd-disagg"},
	}
	if _, err := registry.ValidateIntent(request, artifact); err != nil {
		t.Fatalf("validate intent: %v", err)
	}
	request.Runtime = RuntimeIntent{Backend: "sglang", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-sglang-dgd-disagg"}
	if _, err := registry.ValidateIntent(request, artifact); err != nil {
		t.Fatalf("validate sglang intent: %v", err)
	}
	request.Runtime.Recipe = "missing"
	if _, err := registry.ValidateIntent(request, artifact); err == nil {
		t.Fatal("expected unsupported recipe error")
	}
}

func TestRecipeRegistryRejectsInvalidIntent(t *testing.T) {
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	artifact := ModelArtifact{Family: "deepseek-v4", Variant: "flash", Quantization: "fp8"}
	request := CreateServingApplicationRequest{
		Model:   ModelIntent{Family: "deepseek-v4", Variant: "flash", ArtifactID: "artifact-1", Quantization: "fp8"},
		Runtime: RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-sglang-dgd-disagg"},
	}
	if _, err := registry.ValidateIntent(request, artifact); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected backend mismatch to be invalid input, got %v", err)
	}

	request.Runtime = RuntimeIntent{Backend: "sglang", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-sglang-dgd-disagg"}
	artifact.Quantization = "int4"
	if _, err := registry.ValidateIntent(request, artifact); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected quantization mismatch to be invalid input, got %v", err)
	}
}

func findCreationPlan(plans []ServingApplicationCreationPlan, recipeID string) *ServingApplicationCreationPlan {
	for _, plan := range plans {
		if plan.Recipe.Metadata.ID == recipeID {
			copy := plan
			return &copy
		}
	}
	return nil
}

func TestRecipeRegistryConfigMapOverride(t *testing.T) {
	overrideDir := t.TempDir()
	contents := `apiVersion: inference.zhiliu.dev/v1alpha1
kind: ServingRecipe
metadata:
  id: deepseek-v4-flash-vllm-dgd-disagg
  name: Override
spec:
  model:
    family: deepseek-v4
    variants: [flash]
    quantizations: [fp8]
  runtime:
    backend: vllm
    topology: pd-disagg
  support:
    status: blocked
    warning: temporarily disabled
  template:
    path: deployment/examples/deepseek-v4-flash-vllm-dgd-disagg.yaml
    renderer: string-replacement-v1
`
	if err := os.WriteFile(filepath.Join(overrideDir, "recipe.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := LoadRecipeRegistry("config/recipes", overrideDir)
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	recipe, ok := registry.Get("deepseek-v4-flash-vllm-dgd-disagg")
	if !ok || recipe.Source != "configmap" || recipe.Spec.Support.Status != RecipeSupportStatusBlocked {
		t.Fatalf("expected configmap override, got %+v", recipe)
	}
	artifact := ModelArtifact{Family: "deepseek-v4", Variant: "flash", Quantization: "fp8"}
	request := CreateServingApplicationRequest{
		Model:   ModelIntent{Family: "deepseek-v4", Variant: "flash", ArtifactID: "artifact-1", Quantization: "fp8"},
		Runtime: RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-vllm-dgd-disagg"},
	}
	if _, err := registry.ValidateIntent(request, artifact); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected blocked override to be invalid input, got %v", err)
	}
}
