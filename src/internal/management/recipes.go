package management

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RecipeSupportStatus string

const (
	RecipeSupportStatusSupported    RecipeSupportStatus = "supported"
	RecipeSupportStatusExperimental RecipeSupportStatus = "experimental"
	RecipeSupportStatusBlocked      RecipeSupportStatus = "blocked"
)

type ServingRecipe struct {
	APIVersion string                `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                `json:"kind" yaml:"kind"`
	Metadata   ServingRecipeMetadata `json:"metadata" yaml:"metadata"`
	Spec       ServingRecipeSpec     `json:"spec" yaml:"spec"`
	Source     string                `json:"source,omitempty" yaml:"-"`
	LoadedAt   time.Time             `json:"loadedAt,omitempty" yaml:"-"`
}

type ServingRecipeMetadata struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description"`
}

type ServingRecipeSpec struct {
	Model    ServingRecipeModel    `json:"model" yaml:"model"`
	Runtime  ServingRecipeRuntime  `json:"runtime" yaml:"runtime"`
	Support  ServingRecipeSupport  `json:"support" yaml:"support"`
	Template ServingRecipeTemplate `json:"template" yaml:"template"`
	Defaults ServingRecipeDefaults `json:"defaults" yaml:"defaults"`
}

type ServingRecipeModel struct {
	Family        string   `json:"family" yaml:"family"`
	Variants      []string `json:"variants" yaml:"variants"`
	Quantizations []string `json:"quantizations" yaml:"quantizations"`
}

type ServingRecipeRuntime struct {
	Backend  string `json:"backend" yaml:"backend"`
	Topology string `json:"topology" yaml:"topology"`
}

type ServingRecipeSupport struct {
	Status  RecipeSupportStatus `json:"status" yaml:"status"`
	Warning string              `json:"warning,omitempty" yaml:"warning"`
	Reason  string              `json:"reason,omitempty" yaml:"reason"`
}

type ServingRecipeTemplate struct {
	Path     string `json:"path" yaml:"path"`
	Renderer string `json:"renderer" yaml:"renderer"`
}

type ServingRecipeDefaults struct {
	Namespace          string `json:"namespace,omitempty" yaml:"namespace"`
	Protocol           string `json:"protocol,omitempty" yaml:"protocol"`
	Exposure           string `json:"exposure,omitempty" yaml:"exposure"`
	OptimizationTarget string `json:"optimizationTarget,omitempty" yaml:"optimizationTarget"`
	ProfilingMode      string `json:"profilingMode,omitempty" yaml:"profilingMode"`
}

type RecipeRegistry struct {
	recipes map[string]ServingRecipe
}

func LoadRecipeRegistry(builtinDir string, overrideDir string) (*RecipeRegistry, error) {
	registry := &RecipeRegistry{recipes: map[string]ServingRecipe{}}
	if err := registry.loadDir(builtinDir, "builtin"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(overrideDir) != "" {
		if err := registry.loadDir(overrideDir, "configmap"); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func MustLoadDefaultRecipeRegistry() *RecipeRegistry {
	registry, err := LoadRecipeRegistry(defaultRecipeDir(), os.Getenv("RECIPE_DIR"))
	if err != nil {
		return &RecipeRegistry{recipes: map[string]ServingRecipe{}}
	}
	return registry
}

func (r *RecipeRegistry) List() []ServingRecipe {
	if r == nil {
		return nil
	}
	recipes := make([]ServingRecipe, 0, len(r.recipes))
	for _, recipe := range r.recipes {
		recipes = append(recipes, recipe)
	}
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].Metadata.ID < recipes[j].Metadata.ID })
	return recipes
}

func (r *RecipeRegistry) Get(id string) (ServingRecipe, bool) {
	if r == nil {
		return ServingRecipe{}, false
	}
	recipe, ok := r.recipes[strings.TrimSpace(id)]
	return recipe, ok
}

func (r *RecipeRegistry) ValidateIntent(req CreateServingApplicationRequest, artifact ModelArtifact) (ServingRecipe, error) {
	recipe, ok := r.Get(req.Runtime.Recipe)
	if !ok {
		return ServingRecipe{}, fmt.Errorf("%w: unsupported recipe", ErrInvalidInput)
	}
	if recipe.Spec.Support.Status == RecipeSupportStatusBlocked {
		message := strings.TrimSpace(recipe.Spec.Support.Warning)
		if message == "" {
			message = strings.TrimSpace(recipe.Spec.Support.Reason)
		}
		if message == "" {
			message = "recipe is blocked"
		}
		return ServingRecipe{}, fmt.Errorf("%w: %s", ErrInvalidInput, message)
	}
	if !recipeMatchesIntent(recipe, req, artifact) {
		return ServingRecipe{}, fmt.Errorf("%w: recipe does not match model artifact or runtime intent", ErrInvalidInput)
	}
	return recipe, nil
}

func (r *RecipeRegistry) loadDir(dir string, source string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("recipe directory is required")
	}
	resolved, err := resolveExistingPath(dir)
	if err != nil {
		if source == "configmap" && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("recipe dir %s: %w", dir, err)
	}
	paths, err := filepath.Glob(filepath.Join(resolved, "*.yaml"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var recipe ServingRecipe
		if err := yaml.Unmarshal(contents, &recipe); err != nil {
			return fmt.Errorf("parse recipe %s: %w", path, err)
		}
		recipe.Source = source
		recipe.LoadedAt = time.Now().UTC()
		if err := validateRecipe(recipe); err != nil {
			return fmt.Errorf("validate recipe %s: %w", path, err)
		}
		r.recipes[recipe.Metadata.ID] = recipe
	}
	return nil
}

func validateRecipe(recipe ServingRecipe) error {
	if recipe.APIVersion != "inference.zhiliu.dev/v1alpha1" || recipe.Kind != "ServingRecipe" {
		return fmt.Errorf("apiVersion and kind must be inference.zhiliu.dev/v1alpha1 ServingRecipe")
	}
	if strings.TrimSpace(recipe.Metadata.ID) == "" || strings.TrimSpace(recipe.Metadata.Name) == "" {
		return fmt.Errorf("metadata.id and metadata.name are required")
	}
	if strings.TrimSpace(recipe.Spec.Model.Family) == "" || len(recipe.Spec.Model.Variants) == 0 || len(recipe.Spec.Model.Quantizations) == 0 {
		return fmt.Errorf("spec.model family, variants, and quantizations are required")
	}
	if strings.TrimSpace(recipe.Spec.Runtime.Backend) == "" || strings.TrimSpace(recipe.Spec.Runtime.Topology) == "" {
		return fmt.Errorf("spec.runtime backend and topology are required")
	}
	switch recipe.Spec.Support.Status {
	case RecipeSupportStatusSupported, RecipeSupportStatusExperimental, RecipeSupportStatusBlocked:
	default:
		return fmt.Errorf("spec.support.status must be supported, experimental, or blocked")
	}
	if strings.TrimSpace(recipe.Spec.Template.Path) == "" || strings.TrimSpace(recipe.Spec.Template.Renderer) == "" {
		return fmt.Errorf("spec.template path and renderer are required")
	}
	if recipe.Spec.Template.Renderer != RecipeRendererStringReplacementV1 {
		return fmt.Errorf("spec.template.renderer must be %s", RecipeRendererStringReplacementV1)
	}
	if _, err := resolveExistingPath(recipe.Spec.Template.Path); err != nil {
		return fmt.Errorf("template.path %s: %w", recipe.Spec.Template.Path, err)
	}
	return nil
}

func recipeMatchesIntent(recipe ServingRecipe, req CreateServingApplicationRequest, artifact ModelArtifact) bool {
	return recipe.Spec.Model.Family == req.Model.Family &&
		recipe.Spec.Model.Family == artifact.Family &&
		containsString(recipe.Spec.Model.Variants, req.Model.Variant) &&
		containsString(recipe.Spec.Model.Variants, artifact.Variant) &&
		containsString(recipe.Spec.Model.Quantizations, req.Model.Quantization) &&
		containsString(recipe.Spec.Model.Quantizations, artifact.Quantization) &&
		recipe.Spec.Runtime.Backend == req.Runtime.Backend &&
		recipe.Spec.Runtime.Topology == req.Runtime.Topology
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func defaultRecipeDir() string {
	if value := os.Getenv("BUILTIN_RECIPE_DIR"); strings.TrimSpace(value) != "" {
		return value
	}
	return "config/recipes"
}

func resolveExistingPath(path string) (string, error) {
	candidates := []string{
		path,
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
		filepath.Join("..", "..", "..", path),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}
