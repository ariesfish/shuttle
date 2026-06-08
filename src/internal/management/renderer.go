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

const deepSeekV4FlashVLLMDGDDisaggTemplate = "../deployment/examples/deepseek-v4-flash-vllm-dgd-disagg.yaml"

func RenderKnownTemplate(app ServingApplication, artifact ModelArtifact) (RenderedManifest, error) {
	if app.Model.Family != "deepseek-v4" || app.Model.Variant != "flash" || app.Runtime.Backend != "vllm" || app.Runtime.Topology != "pd-disagg" {
		return RenderedManifest{}, fmt.Errorf("%w: unsupported serving application template", ErrInvalidInput)
	}
	if app.Runtime.Recipe != "deepseek-v4-flash-vllm-dgd-disagg" {
		return RenderedManifest{}, fmt.Errorf("%w: unsupported recipe", ErrInvalidInput)
	}

	content, err := readTemplate(deepSeekV4FlashVLLMDGDDisaggTemplate)
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
		"name: dsv4-disagg-dgd":         "name: " + resourceName,
		"namespace: dynamo-system":      "namespace: " + app.Placement.Namespace,
		"deepseek-ai/DeepSeek-V4-Flash": modelName,
		"/home/dynamo/.cache/huggingface/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/6976c7ff1b30a1b2cb7805021b8ba4684041f136": modelPath,
		"path: \"/data/cache/hub\"": "path: \"" + hostCachePath + "\"",
	}
	for oldText, newText := range replacements {
		content = strings.ReplaceAll(content, oldText, newText)
	}

	return RenderedManifest{Name: resourceName + ".yaml", Content: content}, nil
}

func readTemplate(relativePath string) (string, error) {
	candidates := []string{
		relativePath,
		filepath.Join("..", relativePath),
		filepath.Join("..", "..", relativePath),
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
