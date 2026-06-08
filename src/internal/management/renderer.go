package management

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RenderedManifest struct {
	Name    string
	Content string
}

func RenderKnownTemplate(app ServingApplication, artifact ModelArtifact) (RenderedManifest, error) {
	registry := MustLoadDefaultRecipeRegistry()
	recipe, ok := registry.Get(app.Runtime.Recipe)
	if !ok {
		return RenderedManifest{}, fmt.Errorf("%w: unsupported recipe", ErrInvalidInput)
	}
	return RenderRecipeTemplate(recipe, app, artifact)
}

func RenderRecipeTemplate(recipe ServingRecipe, app ServingApplication, artifact ModelArtifact) (RenderedManifest, error) {
	if recipe.Spec.Template.Renderer != "string-replacement-v1" {
		return RenderedManifest{}, fmt.Errorf("%w: unsupported recipe renderer", ErrInvalidInput)
	}
	if recipe.Metadata.ID != app.Runtime.Recipe {
		return RenderedManifest{}, fmt.Errorf("%w: recipe does not match serving application", ErrInvalidInput)
	}
	if recipe.Spec.Model.Family != app.Model.Family || !containsString(recipe.Spec.Model.Variants, app.Model.Variant) || recipe.Spec.Runtime.Backend != app.Runtime.Backend || recipe.Spec.Runtime.Topology != app.Runtime.Topology {
		return RenderedManifest{}, fmt.Errorf("%w: recipe does not match serving application template", ErrInvalidInput)
	}

	content, err := readTemplate(recipe.Spec.Template.Path)
	if err != nil {
		return RenderedManifest{}, err
	}
	modelName := servedModelName(artifact)
	modelPath := strings.TrimRight(artifact.PVCMountPath, "/") + "/" + strings.TrimLeft(artifact.PVCModelPath, "/")
	hostCachePath := artifact.HostCachePath
	if strings.TrimSpace(hostCachePath) == "" {
		hostCachePath = "/data/cache/hub"
	}
	resourceName := kubernetesName(app.Name)
	if resourceName == "" {
		resourceName = kubernetesName(app.ID)
	}

	replacements := map[string]string{
		"name: " + templateResourceName(content):                    "name: " + resourceName,
		"namespace: dynamo-system":                                  "namespace: " + app.Placement.Namespace,
		"inference.aistudio.dev/serving-application: dsv4-template": "inference.aistudio.dev/serving-application: " + resourceName,
		"deepseek-ai/DeepSeek-V4-Flash":                             modelName,
		"/home/dynamo/.cache/huggingface/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/6976c7ff1b30a1b2cb7805021b8ba4684041f136": modelPath,
		"path: \"/data/cache/hub\"": "path: \"" + hostCachePath + "\"",
	}
	for oldText, newText := range replacements {
		content = strings.ReplaceAll(content, oldText, newText)
	}

	return RenderedManifest{Name: resourceName + ".yaml", Content: content}, nil
}

func templateResourceName(content string) string {
	lines := strings.Split(content, "\n")
	inMetadata := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "metadata:" {
			inMetadata = true
			continue
		}
		if inMetadata && strings.HasPrefix(line, "spec:") {
			return ""
		}
		if inMetadata && strings.HasPrefix(trimmed, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		}
	}
	return ""
}

func readTemplate(relativePath string) (string, error) {
	candidates := []string{
		relativePath,
		filepath.Join("..", relativePath),
		filepath.Join("..", "..", relativePath),
		filepath.Join("..", "..", "..", relativePath),
	}
	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate)
		if err == nil {
			return string(content), nil
		}
	}
	return "", fmt.Errorf("read template %s: not found", relativePath)
}

func servedModelName(artifact ModelArtifact) string {
	switch artifact.Family + ":" + artifact.Variant {
	case "deepseek-v4:flash":
		return "deepseek-ai/DeepSeek-V4-Flash"
	case "deepseek-v4:pro":
		return "deepseek-ai/DeepSeek-V4-Pro"
	default:
		return artifact.Family + "/" + artifact.Variant
	}
}

func kubernetesName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
