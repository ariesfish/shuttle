package management

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

const RecipeRendererStringReplacementV1 = "string-replacement-v1"

type TemplateBindings struct {
	ResourceName  string
	Namespace     string
	ModelName     string
	ModelPath     string
	HostCachePath string
}

type ServingRecipeRenderPlan struct {
	RecipeID     string
	Renderer     string
	TemplatePath string
	Bindings     TemplateBindings
}

func RenderRecipeTemplate(recipe ServingRecipe, app ServingApplication, artifact ModelArtifact) (RenderedManifest, error) {
	plan, err := NewServingRecipeRenderPlan(recipe, app, artifact)
	if err != nil {
		return RenderedManifest{}, err
	}
	content, err := readTemplate(plan.TemplatePath)
	if err != nil {
		return RenderedManifest{}, err
	}
	content, err = plan.Render(content)
	if err != nil {
		return RenderedManifest{}, err
	}
	return RenderedManifest{Name: plan.Bindings.ResourceName + ".yaml", Content: content}, nil
}

func NewServingRecipeRenderPlan(recipe ServingRecipe, app ServingApplication, artifact ModelArtifact) (ServingRecipeRenderPlan, error) {
	if recipe.Spec.Template.Renderer != RecipeRendererStringReplacementV1 {
		return ServingRecipeRenderPlan{}, fmt.Errorf("%w: unsupported recipe renderer", ErrInvalidInput)
	}
	if recipe.Metadata.ID != app.Runtime.Recipe {
		return ServingRecipeRenderPlan{}, fmt.Errorf("%w: recipe does not match serving application", ErrInvalidInput)
	}
	if recipe.Spec.Model.Family != app.Model.Family || !containsString(recipe.Spec.Model.Variants, app.Model.Variant) || recipe.Spec.Runtime.Backend != app.Runtime.Backend || recipe.Spec.Runtime.Topology != app.Runtime.Topology {
		return ServingRecipeRenderPlan{}, fmt.Errorf("%w: recipe does not match serving application template", ErrInvalidInput)
	}
	return ServingRecipeRenderPlan{
		RecipeID:     recipe.Metadata.ID,
		Renderer:     recipe.Spec.Template.Renderer,
		TemplatePath: recipe.Spec.Template.Path,
		Bindings:     NewTemplateBindings(app, artifact),
	}, nil
}

func (p ServingRecipeRenderPlan) Render(content string) (string, error) {
	switch p.Renderer {
	case RecipeRendererStringReplacementV1:
		return renderStringReplacementV1(content, p.Bindings)
	default:
		return "", fmt.Errorf("%w: unsupported recipe renderer", ErrInvalidInput)
	}
}

func NewTemplateBindings(app ServingApplication, artifact ModelArtifact) TemplateBindings {
	modelPath := strings.TrimRight(artifact.PVCMountPath, "/") + "/" + strings.TrimLeft(artifact.PVCModelPath, "/")
	hostCachePath := strings.TrimSpace(artifact.HostCachePath)
	if hostCachePath == "" {
		hostCachePath = "/data/cache/hub"
	}
	resourceName := kubernetesName(app.Name)
	if resourceName == "" {
		resourceName = kubernetesName(app.ID)
	}
	return TemplateBindings{
		ResourceName:  resourceName,
		Namespace:     strings.TrimSpace(app.Placement.Namespace),
		ModelName:     servedModelName(artifact),
		ModelPath:     modelPath,
		HostCachePath: hostCachePath,
	}
}

func renderStringReplacementV1(content string, bindings TemplateBindings) (string, error) {
	// Prefer explicit variables in new templates. Keep literal replacements so
	// existing Git-managed Serving Recipes continue to render without mutating
	// historical deployment examples.
	if strings.Contains(content, "{{") {
		rendered, err := renderExplicitTemplate(content, bindings)
		if err != nil {
			return "", err
		}
		content = rendered
	}

	for _, replacement := range legacyTemplateReplacements(content, bindings) {
		if strings.TrimSpace(replacement.oldText) == "name:" {
			continue
		}
		content = strings.ReplaceAll(content, replacement.oldText, replacement.newText)
	}
	return content, nil
}

func renderExplicitTemplate(content string, bindings TemplateBindings) (string, error) {
	parsed, err := template.New("serving-recipe").Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse serving recipe template: %w", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, bindings); err != nil {
		return "", fmt.Errorf("render serving recipe template: %w", err)
	}
	return rendered.String(), nil
}

type templateReplacement struct {
	oldText string
	newText string
}

func legacyTemplateReplacements(content string, bindings TemplateBindings) []templateReplacement {
	return []templateReplacement{
		{oldText: "name: " + templateResourceName(content), newText: "name: " + bindings.ResourceName},
		{oldText: "namespace: dynamo-system", newText: "namespace: " + bindings.Namespace},
		{oldText: "inference.zhiliu.dev/serving-application: dsv4-template", newText: "inference.zhiliu.dev/serving-application: " + bindings.ResourceName},
		{oldText: "deepseek-ai/DeepSeek-V4-Flash", newText: bindings.ModelName},
		{oldText: "/home/dynamo/.cache/huggingface/models--deepseek-ai--DeepSeek-V4-Flash/snapshots/6976c7ff1b30a1b2cb7805021b8ba4684041f136", newText: bindings.ModelPath},
		{oldText: "path: \"/data/cache/hub\"", newText: "path: \"" + bindings.HostCachePath + "\""},
	}
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
