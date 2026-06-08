package management

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderRecipeSnapshots(t *testing.T) {
	registry, err := LoadRecipeRegistry("config/recipes", "")
	if err != nil {
		t.Fatalf("load recipes: %v", err)
	}
	for _, recipe := range registry.List() {
		recipe := recipe
		t.Run(recipe.Metadata.ID, func(t *testing.T) {
			app := snapshotServingApplication(recipe)
			artifact := snapshotModelArtifact()
			manifest, err := RenderRecipeTemplate(recipe, app, artifact)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			snapshotPath := filepath.Join("testdata", "render-snapshots", recipe.Metadata.ID+".yaml")
			if os.Getenv("UPDATE_RENDER_SNAPSHOTS") == "1" {
				if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(snapshotPath, []byte(manifest.Content), 0o600); err != nil {
					t.Fatal(err)
				}
				return
			}
			expected, err := os.ReadFile(snapshotPath)
			if err != nil {
				t.Fatalf("read snapshot: %v", err)
			}
			if string(expected) != manifest.Content {
				t.Fatalf("render snapshot mismatch for %s; run UPDATE_RENDER_SNAPSHOTS=1 go test ./internal/management -run TestRenderRecipeSnapshots", recipe.Metadata.ID)
			}
		})
	}
}

func snapshotServingApplication(recipe ServingRecipe) ServingApplication {
	return ServingApplication{
		ID:   "app-1",
		Name: recipe.Metadata.Name,
		Model: ModelIntent{
			Family:       recipe.Spec.Model.Family,
			Variant:      recipe.Spec.Model.Variants[0],
			ArtifactID:   "artifact-1",
			Quantization: recipe.Spec.Model.Quantizations[0],
		},
		Placement: PlacementIntent{Namespace: "tenant-a"},
		Runtime: RuntimeIntent{
			Backend:  recipe.Spec.Runtime.Backend,
			Topology: recipe.Spec.Runtime.Topology,
			Recipe:   recipe.Metadata.ID,
		},
	}
}

func snapshotModelArtifact() ModelArtifact {
	return ModelArtifact{
		Family:        "deepseek-v4",
		Variant:       "flash",
		PVCMountPath:  "/models",
		PVCModelPath:  "hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1",
		HostCachePath: "/data/models/hub",
		Quantization:  "fp8",
	}
}
