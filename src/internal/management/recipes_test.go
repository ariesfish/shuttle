package management

import (
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
}
