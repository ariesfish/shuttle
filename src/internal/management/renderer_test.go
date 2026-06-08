package management

import (
	"strings"
	"testing"
)

func TestRenderKnownTemplateForDeepSeekV4Flash(t *testing.T) {
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

	manifest, err := RenderKnownTemplate(app, artifact)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	assertContains(t, manifest.Content, "name: deepseek-v4-flash")
	assertContains(t, manifest.Content, "namespace: tenant-a")
	assertContains(t, manifest.Content, "/models/hub/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1")
	assertContains(t, manifest.Content, "path: \"/data/models/hub\"")
}

func assertContains(t *testing.T, value, substring string) {
	t.Helper()
	if !strings.Contains(value, substring) {
		t.Fatalf("expected rendered manifest to contain %q", substring)
	}
}
