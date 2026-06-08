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

	"zhiliu/internal/management"
	platformtask "zhiliu/internal/task"
)

type Executor interface {
	Execute(ctx context.Context, task management.Task) (map[string]any, error)
}

type TaskExecutor struct {
	dryRunner   ManifestDryRunner
	applier     ManifestApplier
	watcher     ResourceWatcher
	deleter     ResourceDeleter
	diagnostics DiagnosticsCollector
	now         func() time.Time
}

type FakeKubernetesExecutor struct{}

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
	return &TaskExecutor{dryRunner: dryRunner, applier: applier, watcher: watcher, deleter: deleter, diagnostics: KubectlDiagnosticsCollector{}, now: time.Now}
}

func (FakeKubernetesExecutor) Execute(_ context.Context, task management.Task) (map[string]any, error) {
	switch task.Type {
	case management.TaskTypePreviewDeploymentDiff, management.TaskTypeApplyDeployment, management.TaskTypeDeleteBeforeApply, management.TaskTypeRetireDeployment, management.TaskTypeFetchDiagnostics:
	default:
		return map[string]any{"mode": "fake-noop", "taskType": task.Type}, nil
	}
	payload, err := platformtask.DecodePayload(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload})
	if err != nil {
		return nil, err
	}
	resourceRef := agentResourceRef(payload.Resource())
	switch value := payload.(type) {
	case platformtask.RenderedDeploymentPayload:
		if task.Type == management.TaskTypePreviewDeploymentDiff {
			return map[string]any{"mode": "fake-server-side-dry-run", "manifestCount": len(value.Manifests()), "stdout": "fake dry-run ok", "phase": "Validated"}, nil
		}
		if task.Type == management.TaskTypeDeleteBeforeApply {
			return map[string]any{"mode": "fake-delete-before-apply", "resource": resourceRef.Name, "namespace": resourceRef.Namespace, "endpointUrl": endpointURLFromPayload(resourceRef, nil), "phase": "Ready", "deletedBeforeApply": true, "message": "fake redeploy ok"}, nil
		}
		return map[string]any{"mode": "fake-apply", "resource": resourceRef.Name, "namespace": resourceRef.Namespace, "endpointUrl": endpointURLFromPayload(resourceRef, nil), "phase": "Ready", "message": "fake apply ok"}, nil
	case platformtask.ResourcePayload:
		if task.Type == management.TaskTypeRetireDeployment {
			return map[string]any{"mode": "fake-retire", "resource": resourceRef.Name, "namespace": resourceRef.Namespace, "deleted": true, "message": "fake retire ok"}, nil
		}
		return map[string]any{"mode": "fake-diagnostics", "resource": resourceRef.Name, "namespace": resourceRef.Namespace, "sections": []any{}}, nil
	default:
		return map[string]any{"mode": "fake-noop", "taskType": task.Type}, nil
	}
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
	case management.TaskTypeFetchDiagnostics:
		return e.fetchDiagnostics(ctx, task)
	default:
		return map[string]any{
			"mode":      "noop",
			"taskType":  task.Type,
			"handledAt": e.now().UTC().Format(time.RFC3339),
		}, nil
	}
}

func (e *TaskExecutor) previewDeploymentDiff(ctx context.Context, task management.Task) (map[string]any, error) {
	payload, err := renderedDeploymentPayload(task)
	if err != nil {
		return nil, err
	}
	manifests := agentManifests(payload.Manifests())
	result, err := e.dryRunner.ServerSideDryRun(ctx, manifests)
	if err != nil {
		return nil, err
	}
	return platformtask.EncodeResult(platformtask.PreviewResult{
		ManifestCount: len(manifests),
		Stdout:        result.Stdout,
		Stderr:        result.Stderr,
		HandledAt:     e.now().UTC(),
	}), nil
}

func renderedDeploymentPayload(task management.Task) (platformtask.RenderedDeploymentPayload, error) {
	payload, err := platformtask.DecodePayload(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload})
	if err != nil {
		return platformtask.RenderedDeploymentPayload{}, err
	}
	rendered, ok := payload.(platformtask.RenderedDeploymentPayload)
	if !ok {
		return platformtask.RenderedDeploymentPayload{}, fmt.Errorf("%w: expected rendered deployment payload", platformtask.ErrInvalidPayload)
	}
	return rendered, nil
}

func resourcePayload(task management.Task) (platformtask.ResourcePayload, error) {
	payload, err := platformtask.DecodePayload(platformtask.DTO{ID: task.ID, ClusterID: task.ClusterID, Type: platformtask.Type(task.Type), Payload: task.Payload})
	if err != nil {
		return platformtask.ResourcePayload{}, err
	}
	resource, ok := payload.(platformtask.ResourcePayload)
	if !ok {
		return platformtask.ResourcePayload{}, fmt.Errorf("%w: expected resource payload", platformtask.ErrInvalidPayload)
	}
	return resource, nil
}

func agentManifests(manifests []platformtask.Manifest) []Manifest {
	output := make([]Manifest, 0, len(manifests))
	for _, manifest := range manifests {
		output = append(output, Manifest{Name: manifest.Name, Content: manifest.Content})
	}
	return output
}

func agentResourceRef(ref platformtask.ResourceRef) ResourceRef {
	return ResourceRef{Name: ref.Name, Namespace: ref.Namespace}
}

func taskResourceRef(ref ResourceRef) platformtask.ResourceRef {
	return platformtask.ResourceRef{Name: ref.Name, Namespace: ref.Namespace}
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
	payload, err := renderedDeploymentPayload(task)
	if err != nil {
		return nil, err
	}
	return e.applyAndWatch(ctx, agentManifests(payload.Manifests()), agentResourceRef(payload.Resource()), platformtask.TypeApplyDeployment, nil)
}

func (e *TaskExecutor) deleteBeforeApply(ctx context.Context, task management.Task) (map[string]any, error) {
	payload, err := renderedDeploymentPayload(task)
	if err != nil {
		return nil, err
	}
	resourceRef := agentResourceRef(payload.Resource())
	deleteResult, err := e.deleter.DeleteAndWait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	return e.applyAndWatch(ctx, agentManifests(payload.Manifests()), resourceRef, platformtask.TypeDeleteBeforeApply, &deleteResult)
}

func (e *TaskExecutor) retireDeployment(ctx context.Context, task management.Task) (map[string]any, error) {
	payload, err := resourcePayload(task)
	if err != nil {
		return nil, err
	}
	resourceRef := agentResourceRef(payload.Resource())
	deleteResult, err := e.deleter.DeleteAndWait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	return platformtask.EncodeResult(platformtask.RetireResult{
		Resource:  payload.Resource(),
		Deleted:   deleteResult.Deleted,
		Message:   deleteResult.Message,
		HandledAt: e.now().UTC(),
	}), nil
}

func (e *TaskExecutor) fetchDiagnostics(ctx context.Context, task management.Task) (map[string]any, error) {
	payload, err := resourcePayload(task)
	if err != nil {
		return nil, err
	}
	resourceRef := agentResourceRef(payload.Resource())
	collector := e.diagnostics
	if collector == nil {
		collector = KubectlDiagnosticsCollector{}
	}
	result, err := collector.Fetch(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	sections := make([]platformtask.DiagnosticsSection, 0, len(result.Sections))
	for _, section := range result.Sections {
		sections = append(sections, platformtask.DiagnosticsSection{Name: section.Name, Output: section.Output, Error: section.Error})
	}
	return platformtask.EncodeResult(platformtask.DiagnosticsResult{
		Resource:  payload.Resource(),
		Sections:  sections,
		HandledAt: e.now().UTC(),
	}), nil
}

func (e *TaskExecutor) applyAndWatch(ctx context.Context, manifests []Manifest, resourceRef ResourceRef, taskType platformtask.Type, deleteResult *DeleteResult) (map[string]any, error) {
	applyResult, err := e.applier.Apply(ctx, manifests)
	if err != nil {
		return nil, err
	}
	watchResult, err := e.watcher.Wait(ctx, resourceRef)
	if err != nil {
		return nil, err
	}
	result := platformtask.DeploymentResult{
		TypeValue:     taskType,
		ManifestCount: len(manifests),
		Stdout:        applyResult.Stdout,
		Stderr:        applyResult.Stderr,
		Resource:      taskResourceRef(resourceRef),
		EndpointURL:   endpointURLFromPayload(resourceRef, manifests),
		Phase:         watchResult.Phase,
		Message:       watchResult.Message,
		HandledAt:     e.now().UTC(),
	}
	if deleteResult != nil {
		result.DeletedBeforeApply = deleteResult.Deleted
		result.DeleteMessage = deleteResult.Message
	}
	return platformtask.EncodeResult(result), nil
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

type DiagnosticsCollector interface {
	Fetch(context.Context, ResourceRef) (DiagnosticsResult, error)
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

type DiagnosticsResult struct {
	Sections []DiagnosticsSection
}

type DiagnosticsSection struct {
	Name   string
	Output string
	Error  string
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
	runKubectl  func(context.Context, string, ...string) (string, error)
}

type KubectlDiagnosticsCollector struct {
	KubectlPath string
	TailLines   int
	MaxBytes    int
	runKubectl  func(context.Context, string, ...string) (string, error)
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

func (c KubectlDiagnosticsCollector) Fetch(ctx context.Context, ref ResourceRef) (DiagnosticsResult, error) {
	if strings.TrimSpace(ref.Name) == "" || strings.TrimSpace(ref.Namespace) == "" {
		return DiagnosticsResult{}, errors.New("resource name and namespace are required")
	}
	kubectl := c.KubectlPath
	if kubectl == "" {
		kubectl = "kubectl"
	}
	runKubectl := c.runKubectl
	if runKubectl == nil {
		runKubectl = runKubectlCombined
	}
	tailLines := c.TailLines
	if tailLines <= 0 {
		tailLines = 200
	}
	maxBytes := c.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 16 * 1024
	}
	selector := "inference.zhiliu.dev/serving-application=" + ref.Name
	dynamoSelector := "nvidia.com/dynamo-graph-deployment-name=" + ref.Name
	commands := []struct {
		name string
		args []string
	}{
		{name: "dynamographdeployment", args: []string{"-n", ref.Namespace, "get", "dynamographdeployment", ref.Name, "-o", "yaml"}},
		{name: "dynamocomponentdeploymentsByLabel", args: []string{"-n", ref.Namespace, "get", "dynamocomponentdeployment", "-l", selector, "-o", "wide"}},
		{name: "dynamocomponentdeploymentByName", args: []string{"-n", ref.Namespace, "get", "dynamocomponentdeployment", ref.Name, "-o", "wide"}},
		{name: "podsByLabel", args: []string{"-n", ref.Namespace, "get", "pod", "-l", selector, "-o", "wide"}},
		{name: "podsByDynamoLabel", args: []string{"-n", ref.Namespace, "get", "pod", "-l", dynamoSelector, "-o", "wide"}},
		{name: "podsByNamePrefix", args: []string{"-n", ref.Namespace, "get", "pod", "-o", "name"}},
		{name: "events", args: []string{"-n", ref.Namespace, "get", "events", "--sort-by=.lastTimestamp"}},
		{name: "currentLogsByLabel", args: []string{"-n", ref.Namespace, "logs", "-l", selector, "--all-containers=true", "--tail", fmt.Sprintf("%d", tailLines), "--prefix=true"}},
		{name: "previousLogsByLabel", args: []string{"-n", ref.Namespace, "logs", "-l", selector, "--all-containers=true", "--previous", "--tail", fmt.Sprintf("%d", tailLines), "--prefix=true"}},
	}
	sections := make([]DiagnosticsSection, 0, len(commands)+2)
	var podNames []string
	for _, command := range commands {
		output, err := runKubectl(ctx, kubectl, command.args...)
		if command.name == "podsByNamePrefix" {
			podNames = podNamesByPrefix(output, ref.Name)
			output = strings.Join(podNames, "\n")
		}
		section := DiagnosticsSection{Name: command.name, Output: truncateBytes(output, maxBytes)}
		if err != nil {
			section.Error = strings.TrimSpace(err.Error())
		}
		sections = append(sections, section)
	}
	sections = append(sections, c.logsForPods(ctx, runKubectl, kubectl, ref.Namespace, podNames, false, tailLines, maxBytes))
	sections = append(sections, c.logsForPods(ctx, runKubectl, kubectl, ref.Namespace, podNames, true, tailLines, maxBytes))
	return DiagnosticsResult{Sections: sections}, nil
}

func (c KubectlDiagnosticsCollector) logsForPods(ctx context.Context, runKubectl func(context.Context, string, ...string) (string, error), kubectl string, namespace string, podNames []string, previous bool, tailLines int, maxBytes int) DiagnosticsSection {
	sectionName := "currentLogsByNamePrefix"
	if previous {
		sectionName = "previousLogsByNamePrefix"
	}
	if len(podNames) == 0 {
		return DiagnosticsSection{Name: sectionName, Output: "no pods matched resource name prefix"}
	}
	var outputs []string
	var sectionErrors []string
	for _, podName := range podNames {
		args := []string{"-n", namespace, "logs", podName, "--all-containers=true"}
		if previous {
			args = append(args, "--previous")
		}
		args = append(args, "--tail", fmt.Sprintf("%d", tailLines), "--prefix=true")
		output, err := runKubectl(ctx, kubectl, args...)
		outputs = append(outputs, "# "+podName+"\n"+output)
		if err != nil {
			sectionErrors = append(sectionErrors, podName+": "+strings.TrimSpace(err.Error()))
		}
	}
	return DiagnosticsSection{Name: sectionName, Output: truncateBytes(strings.Join(outputs, "\n"), maxBytes), Error: strings.Join(sectionErrors, "\n")}
}

func podNamesByPrefix(output string, prefix string) []string {
	var podNames []string
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "pod/")
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		podNames = append(podNames, name)
	}
	return podNames
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
	runKubectl := d.runKubectl
	if runKubectl == nil {
		runKubectl = runKubectlCombined
	}
	deleteOut, err := runKubectl(ctx, kubectl, "-n", ref.Namespace, "delete", "dynamographdeployment", ref.Name, "--ignore-not-found", "--wait=true", "--timeout=120s")
	if err != nil {
		return DeleteResult{}, fmt.Errorf("kubectl delete dynamographdeployment failed: %w: %s", err, strings.TrimSpace(deleteOut))
	}

	labelKey := d.LabelKey
	if labelKey == "" {
		labelKey = "inference.zhiliu.dev/serving-application"
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
		out, err := runKubectl(ctx, kubectl, "-n", ref.Namespace, "delete", cleanup.kind, "-l", selector, "--ignore-not-found", "--wait=true", "--timeout="+cleanup.timeout)
		cleanupMessages = append(cleanupMessages, strings.TrimSpace(out))
		if err != nil {
			return DeleteResult{}, fmt.Errorf("kubectl delete %s by label failed: %w: %s", cleanup.kind, err, strings.TrimSpace(out))
		}
	}
	for _, cleanup := range []struct {
		kind    string
		name    string
		timeout string
	}{
		{kind: "dynamocomponentdeployment", name: ref.Name, timeout: "60s"},
		{kind: "deploy", name: ref.Name, timeout: "60s"},
		{kind: "rs", name: ref.Name, timeout: "60s"},
		{kind: "pod", name: ref.Name, timeout: "60s"},
		{kind: "svc", name: ref.Name, timeout: "60s"},
	} {
		out, err := runKubectl(ctx, kubectl, "-n", ref.Namespace, "delete", cleanup.kind, cleanup.name, "--ignore-not-found", "--wait=true", "--timeout="+cleanup.timeout)
		cleanupMessages = append(cleanupMessages, strings.TrimSpace(out))
		if err != nil {
			return DeleteResult{}, fmt.Errorf("kubectl delete %s/%s by exact name failed: %w: %s", cleanup.kind, cleanup.name, err, strings.TrimSpace(out))
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
		getOut, err := runKubectl(waitCtx, kubectl, "-n", ref.Namespace, "get", "dynamographdeployment", ref.Name)
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
	args := []string{"-n", ref.Namespace, "get", "dynamographdeployment", ref.Name, "-o", "jsonpath={.status.phase}{'|'}{.status.state}{'|'}{.status.conditions[-1].type}{'|'}{.status.conditions[-1].status}{'|'}{.status.conditions[-1].message}"}
	cmd := exec.CommandContext(ctx, kubectl, args...)
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
		stderr = string(exitErr.Stderr)
	}
	if err != nil {
		return WatchResult{}, false, fmt.Errorf("kubectl get dynamographdeployment failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	return dynamoGraphDeploymentWatchResult(ref, string(stdout))
}

func dynamoGraphDeploymentWatchResult(ref ResourceRef, output string) (WatchResult, bool, error) {
	parts := strings.SplitN(output, "|", 5)
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	phase := strings.TrimSpace(parts[0])
	state := strings.TrimSpace(parts[1])
	conditionType := strings.TrimSpace(parts[2])
	conditionStatus := strings.TrimSpace(parts[3])
	message := strings.TrimSpace(parts[4])
	status := phase
	if status == "" {
		status = state
	}
	if status == "" && strings.EqualFold(conditionType, "Ready") {
		status = conditionStatus
	}

	switch strings.ToLower(status) {
	case "ready", "running", "deployed", "successful", "success", "true":
		return WatchResult{Phase: status, Message: message}, true, nil
	case "failed", "error", "false":
		if strings.EqualFold(conditionType, "Ready") && strings.EqualFold(conditionStatus, "False") && !isTerminalDynamoState(state) {
			return WatchResult{Phase: status, Message: message}, false, nil
		}
		return WatchResult{Phase: status, Message: message}, true, fmt.Errorf("DynamoGraphDeployment %s/%s failed: %s", ref.Namespace, ref.Name, message)
	default:
		return WatchResult{Phase: status, Message: message}, false, nil
	}
}

func isTerminalDynamoState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "failed", "error":
		return true
	default:
		return false
	}
}

func runKubectlCombined(ctx context.Context, kubectl string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, kubectl, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func truncateBytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes] + "\n...[truncated]"
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
