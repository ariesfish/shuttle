package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"inference-platform/internal/management"
)

type Executor interface {
	Execute(ctx context.Context, task management.Task) (map[string]any, error)
}

type TaskExecutor struct {
	dryRunner ManifestDryRunner
	applier   ManifestApplier
	watcher   ResourceWatcher
	deleter   ResourceDeleter
	now       func() time.Time
}

func NewTaskExecutor(dryRunner ManifestDryRunner) *TaskExecutor {
	return NewTaskExecutorWithApply(dryRunner, nil, nil)
}

func NewTaskExecutorWithApply(dryRunner ManifestDryRunner, applier ManifestApplier, watcher ResourceWatcher) *TaskExecutor {
	return NewTaskExecutorWithKubernetes(dryRunner, applier, watcher, nil)
}

func NewTaskExecutorWithKubernetes(dryRunner ManifestDryRunner, applier ManifestApplier, watcher ResourceWatcher, deleter ResourceDeleter) *TaskExecutor {
	if dryRunner == nil {
		dryRunner = KubectlDryRunner{}
	}
	if applier == nil {
		applier = KubectlApplier{}
	}
	if watcher == nil {
		watcher = KubectlResourceWatcher{Timeout: 10 * time.Minute, Interval: 5 * time.Second}
	}
	if deleter == nil {
		deleter = KubectlResourceDeleter{Timeout: 10 * time.Minute, Interval: 5 * time.Second}
	}
	return &TaskExecutor{dryRunner: dryRunner, applier: applier, watcher: watcher, deleter: deleter, now: time.Now}
}

func (e *TaskExecutor) Execute(ctx context.Context, task management.Task) (map[string]any, error) {
	switch task.Type {
	case management.TaskTypePreviewDeploymentDiff:
		return e.previewDeploymentDiff(ctx, task)
	case management.TaskTypeApplyDeployment:
		return e.applyDeployment(ctx, task)
	case management.TaskTypeDeleteBeforeApply:
		return e.deleteBeforeApply(ctx, task)
	case management.TaskTypeRetireDeployment:
		return e.retireDeployment(ctx, task)
	default:
		return map[string]any{
			"mode":      "noop",
			"taskType":  task.Type,
			"handledAt": e.now().UTC().Format(time.RFC3339),
		}, nil
	}
}

func (e *TaskExecutor) previewDeploymentDiff(ctx context.Context, task management.Task) (map[string]any, error) {
	manifests, err := manifestsFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	result, err := e.dryRunner.ServerSideDryRun(ctx, manifests)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mode":          "server-side-dry-run",
		"manifestCount": len(manifests),
		"stdout":        result.Stdout,
		"stderr":        result.Stderr,
		"handledAt":     e.now().UTC().Format(time.RFC3339),
	}, nil
}

type Manifest struct {
	Name    string
	Content string
}

type DryRunResult struct {
	Stdout string
	Stderr string
}

func (e *TaskExecutor) applyDeployment(ctx context.Context, task management.Task) (map[string]any, error) {
	manifests, err := manifestsFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	resourceRef, err := resourceRefFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	return e.applyAndWatch(ctx, manifests, resourceRef, "apply-and-watch", nil)
}

func (e *TaskExecutor) deleteBeforeApply(ctx context.Context, task management.Task) (map[string]any, error) {
	manifests, err := manifestsFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	resourceRef, err := resourceRefFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	deleteResult, err := e.deleter.DeleteAndWait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	return e.applyAndWatch(ctx, manifests, resourceRef, "delete-before-apply", &deleteResult)
}

func (e *TaskExecutor) retireDeployment(ctx context.Context, task management.Task) (map[string]any, error) {
	resourceRef, err := resourceRefFromPayload(task.Payload)
	if err != nil {
		return nil, err
	}
	deleteResult, err := e.deleter.DeleteAndWait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mode":      "retire",
		"resource":  resourceRef.Name,
		"namespace": resourceRef.Namespace,
		"deleted":   deleteResult.Deleted,
		"message":   deleteResult.Message,
		"handledAt": e.now().UTC().Format(time.RFC3339),
	}, nil
}

func (e *TaskExecutor) applyAndWatch(ctx context.Context, manifests []Manifest, resourceRef ResourceRef, mode string, deleteResult *DeleteResult) (map[string]any, error) {
	applyResult, err := e.applier.Apply(ctx, manifests)
	if err != nil {
		return nil, err
	}
	watchResult, err := e.watcher.Wait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"mode":          mode,
		"manifestCount": len(manifests),
		"stdout":        applyResult.Stdout,
		"stderr":        applyResult.Stderr,
		"resource":      resourceRef.Name,
		"namespace":     resourceRef.Namespace,
		"endpointUrl":   endpointURLFromPayload(resourceRef, manifests),
		"phase":         watchResult.Phase,
		"message":       watchResult.Message,
		"handledAt":     e.now().UTC().Format(time.RFC3339),
	}
	if deleteResult != nil {
		result["deletedBeforeApply"] = deleteResult.Deleted
		result["deleteMessage"] = deleteResult.Message
	}
	return result, nil
}

type ManifestDryRunner interface {
	ServerSideDryRun(context.Context, []Manifest) (DryRunResult, error)
}

type ManifestApplier interface {
	Apply(context.Context, []Manifest) (ApplyResult, error)
}

type ResourceWatcher interface {
	Wait(context.Context, ResourceRef) (WatchResult, error)
}

type ResourceDeleter interface {
	DeleteAndWait(context.Context, ResourceRef) (DeleteResult, error)
}

type ApplyResult struct {
	Stdout string
	Stderr string
}

type ResourceRef struct {
	Namespace string
	Name      string
}

type WatchResult struct {
	Phase   string
	Message string
}

type DeleteResult struct {
	Deleted bool
	Message string
}

type KubectlDryRunner struct {
	KubectlPath string
	Namespace   string
}

type KubectlApplier struct {
	KubectlPath string
	Namespace   string
}

type KubectlResourceWatcher struct {
	KubectlPath string
	Timeout     time.Duration
	Interval    time.Duration
}

type KubectlResourceDeleter struct {
	KubectlPath string
	Timeout     time.Duration
	Interval    time.Duration
	LabelKey    string
}

func (r KubectlDryRunner) ServerSideDryRun(ctx context.Context, manifests []Manifest) (DryRunResult, error) {
	if len(manifests) == 0 {
		return DryRunResult{}, errors.New("at least one manifest is required")
	}
	dir, err := writeManifestsToTempDir(manifests)
	if err != nil {
		return DryRunResult{}, err
	}
	defer os.RemoveAll(dir)

	kubectl := r.KubectlPath
	if kubectl == "" {
		kubectl = "kubectl"
	}
	args := []string{"apply", "--dry-run=server", "-f", dir, "-o", "yaml"}
	if strings.TrimSpace(r.Namespace) != "" {
		args = append([]string{"-n", strings.TrimSpace(r.Namespace)}, args...)
	}
	cmd := exec.CommandContext(ctx, kubectl, args...)
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		stderr = string(exitErr.Stderr)
	}
	if err != nil {
		return DryRunResult{Stdout: string(stdout), Stderr: stderr}, fmt.Errorf("kubectl server-side dry-run failed: %w", err)
	}
	return DryRunResult{Stdout: string(stdout), Stderr: stderr}, nil
}

func (a KubectlApplier) Apply(ctx context.Context, manifests []Manifest) (ApplyResult, error) {
	if len(manifests) == 0 {
		return ApplyResult{}, errors.New("at least one manifest is required")
	}
	dir, err := writeManifestsToTempDir(manifests)
	if err != nil {
		return ApplyResult{}, err
	}
	defer os.RemoveAll(dir)

	kubectl := a.KubectlPath
	if kubectl == "" {
		kubectl = "kubectl"
	}
	args := []string{"apply", "-f", dir, "-o", "yaml"}
	if strings.TrimSpace(a.Namespace) != "" {
		args = append([]string{"-n", strings.TrimSpace(a.Namespace)}, args...)
	}
	cmd := exec.CommandContext(ctx, kubectl, args...)
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		stderr = string(exitErr.Stderr)
	}
	if err != nil {
		return ApplyResult{Stdout: string(stdout), Stderr: stderr}, fmt.Errorf("kubectl apply failed: %w", err)
	}
	return ApplyResult{Stdout: string(stdout), Stderr: stderr}, nil
}

func (w KubectlResourceWatcher) Wait(ctx context.Context, ref ResourceRef) (WatchResult, error) {
	if strings.TrimSpace(ref.Name) == "" || strings.TrimSpace(ref.Namespace) == "" {
		return WatchResult{}, errors.New("resource name and namespace are required")
	}
	timeout := w.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	interval := w.Interval
	if interval == 0 {
		interval = 5 * time.Second
	}
	watchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		result, done, err := w.getOnce(watchCtx, ref)
		if err != nil {
			return WatchResult{}, err
		}
		if done {
			return result, nil
		}
		select {
		case <-watchCtx.Done():
			return WatchResult{}, fmt.Errorf("wait for DynamoGraphDeployment %s/%s: %w", ref.Namespace, ref.Name, watchCtx.Err())
		case <-ticker.C:
		}
	}
}

func (d KubectlResourceDeleter) DeleteAndWait(ctx context.Context, ref ResourceRef) (DeleteResult, error) {
	if strings.TrimSpace(ref.Name) == "" || strings.TrimSpace(ref.Namespace) == "" {
		return DeleteResult{}, errors.New("resource name and namespace are required")
	}
	kubectl := d.KubectlPath
	if kubectl == "" {
		kubectl = "kubectl"
	}
	deleteOut, err := runKubectlCombined(ctx, kubectl, "-n", ref.Namespace, "delete", "dynamographdeployment", ref.Name, "--ignore-not-found", "--wait=true", "--timeout=120s")
	if err != nil {
		return DeleteResult{}, fmt.Errorf("kubectl delete dynamographdeployment failed: %w: %s", err, strings.TrimSpace(deleteOut))
	}

	labelKey := d.LabelKey
	if labelKey == "" {
		labelKey = "nvidia.com/dynamo-graph-deployment"
	}
	selector := labelKey + "=" + ref.Name
	cleanupMessages := []string{strings.TrimSpace(deleteOut)}
	for _, cleanup := range []struct {
		kind    string
		timeout string
	}{
		{kind: "dynamocomponentdeployment", timeout: "60s"},
		{kind: "pod,deploy,rs,svc", timeout: "60s"},
	} {
		out, err := runKubectlCombined(ctx, kubectl, "-n", ref.Namespace, "delete", cleanup.kind, "-l", selector, "--ignore-not-found", "--wait=true", "--timeout="+cleanup.timeout)
		cleanupMessages = append(cleanupMessages, strings.TrimSpace(out))
		if err != nil {
			return DeleteResult{}, fmt.Errorf("kubectl delete %s by label failed: %w: %s", cleanup.kind, err, strings.TrimSpace(out))
		}
	}

	timeout := d.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	interval := d.Interval
	if interval == 0 {
		interval = 5 * time.Second
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		getOut, err := runKubectlCombined(waitCtx, kubectl, "-n", ref.Namespace, "get", "dynamographdeployment", ref.Name)
		if err != nil && (strings.Contains(getOut, "NotFound") || strings.Contains(getOut, "not found")) {
			return DeleteResult{Deleted: true, Message: strings.Join(nonEmptyStrings(cleanupMessages), "\n")}, nil
		}
		if err != nil {
			return DeleteResult{}, fmt.Errorf("kubectl get dynamographdeployment during delete wait failed: %w: %s", err, strings.TrimSpace(getOut))
		}
		select {
		case <-waitCtx.Done():
			return DeleteResult{}, fmt.Errorf("wait for DynamoGraphDeployment deletion %s/%s: %w", ref.Namespace, ref.Name, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func (w KubectlResourceWatcher) getOnce(ctx context.Context, ref ResourceRef) (WatchResult, bool, error) {
	kubectl := w.KubectlPath
	if kubectl == "" {
		kubectl = "kubectl"
	}
	args := []string{"-n", ref.Namespace, "get", "dynamographdeployment", ref.Name, "-o", "jsonpath={.status.phase}{'|'}{.status.conditions[-1].message}"}
	cmd := exec.CommandContext(ctx, kubectl, args...)
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		stderr = string(exitErr.Stderr)
	}
	if err != nil {
		return WatchResult{}, false, fmt.Errorf("kubectl get dynamographdeployment failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	parts := strings.SplitN(string(stdout), "|", 2)
	phase := strings.TrimSpace(parts[0])
	message := ""
	if len(parts) == 2 {
		message = strings.TrimSpace(parts[1])
	}
	switch strings.ToLower(phase) {
	case "ready", "running", "deployed", "successful", "success":
		return WatchResult{Phase: phase, Message: message}, true, nil
	case "failed", "error":
		return WatchResult{Phase: phase, Message: message}, true, fmt.Errorf("DynamoGraphDeployment %s/%s failed: %s", ref.Namespace, ref.Name, message)
	default:
		return WatchResult{Phase: phase, Message: message}, false, nil
	}
}

func manifestsFromPayload(payload map[string]any) ([]Manifest, error) {
	rawManifests, ok := payload["manifests"]
	if !ok {
		return nil, errors.New("payload.manifests is required")
	}
	items, ok := rawManifests.([]any)
	if !ok {
		return nil, errors.New("payload.manifests must be an array")
	}
	manifests := make([]Manifest, 0, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("payload.manifests[%d] must be an object", index)
		}
		name, _ := object["name"].(string)
		content, _ := object["content"].(string)
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("payload.manifests[%d].content is required", index)
		}
		manifests = append(manifests, Manifest{Name: name, Content: content})
	}
	return manifests, nil
}

func runKubectlCombined(ctx context.Context, kubectl string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, kubectl, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func nonEmptyStrings(values []string) []string {
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			output = append(output, value)
		}
	}
	return output
}

func endpointURLFromPayload(ref ResourceRef, _ []Manifest) string {
	return "http://" + ref.Name + "." + ref.Namespace + ".svc.cluster.local:8000/v1"
}

func resourceRefFromPayload(payload map[string]any) (ResourceRef, error) {
	name, _ := payload["resourceName"].(string)
	namespace, _ := payload["namespace"].(string)
	if strings.TrimSpace(name) == "" || strings.TrimSpace(namespace) == "" {
		return ResourceRef{}, errors.New("payload.resourceName and payload.namespace are required")
	}
	return ResourceRef{Name: strings.TrimSpace(name), Namespace: strings.TrimSpace(namespace)}, nil
}

func writeManifestsToTempDir(manifests []Manifest) (string, error) {
	dir, err := os.MkdirTemp("", "inference-agent-manifests-*")
	if err != nil {
		return "", err
	}
	for index, manifest := range manifests {
		name := sanitizeManifestName(manifest.Name)
		if name == "" {
			name = fmt.Sprintf("manifest-%d.yaml", index+1)
		}
		if filepath.Ext(name) == "" {
			name += ".yaml"
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(manifest.Content), 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

func sanitizeManifestName(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}
