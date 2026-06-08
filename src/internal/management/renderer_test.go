package management

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRecipeTemplateForDeepSeekV4FlashVLLM(t *testing.T) {
	app := ServingApplication{
		ID:   "app-1",
		Name: "DeepSeek V4 Flash",
		Model: ModelIntent{
			Family:       "deepseek-v4",
			Variant:      "flash",
			ArtifactID:   "artifact-1",
			Quantization: "fp8",
		},
		Placement: PlacementIntent{
			Namespace: "tenant-a",
		},
		Runtime: RuntimeIntent{
			Backend:  "vllm",
			Topology: "pd-disagg",
			Recipe:   "deepseek-v4-flash-vllm-dgd-disagg",
		},
	}
	artifact := ModelArtifact{
		Family:        "deepseek-v4",
		Variant:       "flash",
		PVCMountPath:  "/models",
		PVCModelPath:  "hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1",
		HostCachePath: "/data/models/hub",
	}

	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	recipe, ok := registry.Get(app.Runtime.Recipe)
	if !ok {
		t.Fatalf("recipe not found")
	}
	manifest, err := RenderRecipeTemplate(recipe, app, artifact)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertContains(t, manifest.Content, "name: deepseek-v4-flash")
	assertContains(t, manifest.Content, "namespace: tenant-a")
	assertContains(t, manifest.Content, "/models/hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1")
	assertContains(t, manifest.Content, "path: \"/data/models/hub\"")
}

func TestRenderRecipeTemplateForDeepSeekV4FlashSGLang(t *testing.T) {
	app := ServingApplication{
		ID:   "app-1",
		Name: "DeepSeek V4 Flash SGLang",
		Model: ModelIntent{
			Family:       "deepseek-v4",
			Variant:      "flash",
			ArtifactID:   "artifact-1",
			Quantization: "fp8",
		},
		Placement: PlacementIntent{Namespace: "tenant-a"},
		Runtime: RuntimeIntent{
			Backend:  "sglang",
			Topology: "pd-disagg",
			Recipe:   "deepseek-v4-flash-sglang-dgd-disagg",
		},
	}
	artifact := ModelArtifact{
		Family:       "deepseek-v4",
		Variant:      "flash",
		PVCMountPath: "/models",
		PVCModelPath: "hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1",
	}
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	recipe, ok := registry.Get(app.Runtime.Recipe)
	if !ok {
		t.Fatalf("recipe not found")
	}
	manifest, err := RenderRecipeTemplate(recipe, app, artifact)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertContains(t, manifest.Content, "name: deepseek-v4-flash-sglang")
	assertContains(t, manifest.Content, "namespace: tenant-a")
	assertContains(t, manifest.Content, "/models/hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1")
	assertContains(t, manifest.Content, "path: \"/data/cache/hub\"")
}

func TestRenderRecipeTemplateSupportsExplicitVariables(t *testing.T) {
	templatePath := filepath.Join(t.TempDir(), "template.yaml")
	contents := `apiVersion: nvidia.com/v1alpha1
kind: DynamoGraphDeployment
metadata:
  name: {{ .ResourceName }}
  namespace: {{ .Namespace }}
  labels:
    inference.zhiliu.dev/serving-application: {{ .ResourceName }}
spec:
  model: {{ .ModelName }}
  path: {{ .ModelPath }}
  hostCache: {{ .HostCachePath }}
`
	if err := os.WriteFile(templatePath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	app := rendererTestServingApplication("deepseek-v4-flash-vllm-dgd-disagg", "vllm", "DeepSeek V4 Flash")
	artifact := rendererTestModelArtifact()
	recipe := rendererTestRecipe(app.Runtime.Recipe, app.Runtime.Backend, templatePath)
	manifest, err := RenderRecipeTemplate(recipe, app, artifact)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertContains(t, manifest.Content, "name: deepseek-v4-flash")
	assertContains(t, manifest.Content, "namespace: tenant-a")
	assertContains(t, manifest.Content, "model: deepseek-ai/DeepSeek-V4-Flash")
	assertContains(t, manifest.Content, "path: /models/hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1")
	assertContains(t, manifest.Content, "hostCache: /data/models/hub")
}

func TestRenderRecipeTemplateRejectsUnknownExplicitVariable(t *testing.T) {
	templatePath := filepath.Join(t.TempDir(), "template.yaml")
	if err := os.WriteFile(templatePath, []byte("metadata:\n  name: {{ .Missing }}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := rendererTestServingApplication("deepseek-v4-flash-vllm-dgd-disagg", "vllm", "DeepSeek V4 Flash")
	_, err := RenderRecipeTemplate(rendererTestRecipe(app.Runtime.Recipe, app.Runtime.Backend, templatePath), app, rendererTestModelArtifact())
	if err == nil {
		t.Fatal("expected unknown template variable error")
	}
}

func rendererTestServingApplication(recipeID string, backend string, name string) ServingApplication {
	return ServingApplication{
		ID:   "app-1",
		Name: name,
		Model: ModelIntent{
			Family:       "deepseek-v4",
			Variant:      "flash",
			ArtifactID:   "artifact-1",
			Quantization: "fp8",
		},
		Placement: PlacementIntent{Namespace: "tenant-a"},
		Runtime: RuntimeIntent{
			Backend:  backend,
			Topology: "pd-disagg",
			Recipe:   recipeID,
		},
	}
}

func rendererTestModelArtifact() ModelArtifact {
	return ModelArtifact{
		Family:        "deepseek-v4",
		Variant:       "flash",
		PVCMountPath:  "/models",
		PVCModelPath:  "hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1",
		HostCachePath: "/data/models/hub",
		Quantization:  "fp8",
	}
}

func rendererTestRecipe(id string, backend string, templatePath string) ServingRecipe {
	return ServingRecipe{
		APIVersion: "inference.zhiliu.dev/v1alpha1",
		Kind:       "ServingRecipe",
		Metadata:   ServingRecipeMetadata{ID: id, Name: "test recipe"},
		Spec: ServingRecipeSpec{
			Model:    ServingRecipeModel{Family: "deepseek-v4", Variants: []string{"flash"}, Quantizations: []string{"fp8"}},
			Runtime:  ServingRecipeRuntime{Backend: backend, Topology: "pd-disagg"},
			Support:  ServingRecipeSupport{Status: RecipeSupportStatusSupported},
			Template: ServingRecipeTemplate{Path: templatePath, Renderer: RecipeRendererStringReplacementV1},
		},
	}
}

func assertContains(t *testing.T, value, substring string) {
	t.Helper()
	if !strings.Contains(value, substring) {
		t.Fatalf("expected rendered manifest to contain %q", substring)
	}
}
