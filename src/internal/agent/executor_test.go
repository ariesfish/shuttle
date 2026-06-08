package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"inference-platform/internal/management"
)

func TestTaskExecutorPreviewDeploymentDiff(t *testing.T) {
	dryRunner := &fakeDryRunner{result: DryRunResult{Stdout: "kind: List\n", Stderr: "warning"}}
	executor := NewTaskExecutor(dryRunner)

	result, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypePreviewDeploymentDiff,
		Payload: map[string]any{
			"manifests": []any{
				map[string]any{
					"name":    "deepseek.yaml",
					"content": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(dryRunner.manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(dryRunner.manifests))
	}
	if result["mode"] != "server-side-dry-run" || result["manifestCount"] != 1 || result["stdout"] != "kind: List\n" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTaskExecutorApplyDeployment(t *testing.T) {
	applier := &fakeApplier{result: ApplyResult{Stdout: "applied"}}
	watcher := &fakeWatcher{result: WatchResult{Phase: "Ready", Message: "ok"}}
	executor := NewTaskExecutorWithApply(&fakeDryRunner{}, applier, watcher)

	result, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypeApplyDeployment,
		Payload: map[string]any{
			"resourceName": "deepseek-v4-flash",
			"namespace":    "dynamo-system",
			"manifests": []any{map[string]any{
				"name":    "dgd.yaml",
				"content": "kind: DynamoGraphDeployment\n",
			}},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(applier.manifests) != 1 {
		t.Fatalf("expected apply to receive manifest")
	}
	if watcher.ref.Name != "deepseek-v4-flash" || watcher.ref.Namespace != "dynamo-system" {
		t.Fatalf("unexpected watch ref: %+v", watcher.ref)
	}
	if result["mode"] != "apply-and-watch" || result["phase"] != "Ready" || result["endpointUrl"] == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTaskExecutorDeleteBeforeApply(t *testing.T) {
	applier := &fakeApplier{result: ApplyResult{Stdout: "applied"}}
	watcher := &fakeWatcher{result: WatchResult{Phase: "Ready"}}
	deleter := &fakeDeleter{result: DeleteResult{Deleted: true, Message: "deleted"}}
	executor := NewTaskExecutorWithKubernetes(&fakeDryRunner{}, applier, watcher, deleter)

	result, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypeDeleteBeforeApply,
		Payload: map[string]any{
			"resourceName": "deepseek-v4-flash",
			"namespace":    "dynamo-system",
			"manifests": []any{map[string]any{
				"name":    "dgd.yaml",
				"content": "kind: DynamoGraphDeployment\n",
			}},
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !deleter.called || len(applier.manifests) != 1 || watcher.ref.Name != "deepseek-v4-flash" {
		t.Fatalf("expected delete, apply, watch; deleter=%+v applier=%+v watcher=%+v", deleter, applier, watcher)
	}
	if result["mode"] != "delete-before-apply" || result["deletedBeforeApply"] != true {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestKubectlResourceDeleterUsesExactFallbackAfterLabelCleanup(t *testing.T) {
	var calls []string
	deleter := KubectlResourceDeleter{
		Timeout:  1,
		Interval: 1,
		runKubectl: func(_ context.Context, _ string, args ...string) (string, error) {
			call := strings.Join(args, " ")
			calls = append(calls, call)
			if strings.Contains(call, " get dynamographdeployment deepseek-v4-flash") {
				return "Error from server (NotFound): dynamographdeployments.nvidia.com \"deepseek-v4-flash\" not found", errors.New("not found")
			}
			return "ok", nil
		},
	}
	result, err := deleter.DeleteAndWait(context.Background(), ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"})
	if err != nil {
		t.Fatalf("delete and wait: %v", err)
	}
	if !result.Deleted {
		t.Fatalf("expected deleted result: %+v", result)
	}
	assertKubectlCall(t, calls, "delete dynamocomponentdeployment -l inference.zhiliu.dev/serving-application=deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete pod,deploy,rs,svc -l inference.zhiliu.dev/serving-application=deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete dynamocomponentdeployment deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete deploy deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete rs deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete pod deepseek-v4-flash")
	assertKubectlCall(t, calls, "delete svc deepseek-v4-flash")
}

func TestDynamoGraphDeploymentWatchResultUsesStateAndReadyCondition(t *testing.T) {
	ref := ResourceRef{Name: "dsv4", Namespace: "dynamo-system"}
	ready, done, err := dynamoGraphDeploymentWatchResult(ref, "|successful|Ready|True|All resources are ready")
	if err != nil || !done || ready.Phase != "successful" {
		t.Fatalf("expected successful state done, result=%+v done=%v err=%v", ready, done, err)
	}
	pending, done, err := dynamoGraphDeploymentWatchResult(ref, "|pending|Ready|False|Resources not ready")
	if err != nil || done || pending.Phase != "pending" {
		t.Fatalf("expected pending state not done, result=%+v done=%v err=%v", pending, done, err)
	}
	failed, done, err := dynamoGraphDeploymentWatchResult(ref, "|failed|Ready|False|bad")
	if err == nil || !done || failed.Phase != "failed" {
		t.Fatalf("expected failed state terminal error, result=%+v done=%v err=%v", failed, done, err)
	}
}

func TestTaskExecutorFetchDiagnostics(t *testing.T) {
	executor := NewTaskExecutor(&fakeDryRunner{})
	executor.diagnostics = &fakeDiagnosticsCollector{result: DiagnosticsResult{Sections: []DiagnosticsSection{{Name: "pods", Output: "pod ok"}}}}

	result, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypeFetchDiagnostics,
		Payload: map[string]any{
			"resourceName": "deepseek-v4-flash",
			"namespace":    "dynamo-system",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	sections, ok := result["sections"].([]any)
	if !ok || len(sections) != 1 || result["mode"] != "diagnostics" {
		t.Fatalf("unexpected diagnostics result: %+v", result)
	}
}

func TestKubectlDiagnosticsCollectorUsesBoundedCommands(t *testing.T) {
	var calls []string
	collector := KubectlDiagnosticsCollector{
		TailLines: 50,
		MaxBytes:  8,
		runKubectl: func(_ context.Context, _ string, args ...string) (string, error) {
			call := strings.Join(args, " ")
			calls = append(calls, call)
			if strings.Contains(call, "get pod -o name") {
				return "pod/deepseek-v4-flash-abc\npod/other-app", nil
			}
			if strings.Contains(call, "--previous") {
				return "previous log output", errors.New("previous unavailable")
			}
			return "123456789abcdef", nil
		},
	}
	result, err := collector.Fetch(context.Background(), ResourceRef{Name: "deepseek-v4-flash", Namespace: "dynamo-system"})
	if err != nil {
		t.Fatalf("fetch diagnostics: %v", err)
	}
	if len(result.Sections) != 11 {
		t.Fatalf("expected sections, got %+v", result.Sections)
	}
	if !strings.Contains(result.Sections[0].Output, "truncated") || result.Sections[9].Error != "" || result.Sections[10].Error == "" {
		t.Fatalf("expected bounded output and previous-log section error, got %+v", result.Sections)
	}
	assertKubectlCall(t, calls, "get dynamographdeployment deepseek-v4-flash -o yaml")
	assertKubectlCall(t, calls, "get pod -l inference.zhiliu.dev/serving-application=deepseek-v4-flash -o wide")
	assertKubectlCall(t, calls, "get pod -l nvidia.com/dynamo-graph-deployment-name=deepseek-v4-flash -o wide")
	assertKubectlCall(t, calls, "logs -l inference.zhiliu.dev/serving-application=deepseek-v4-flash --all-containers=true --tail 50")
	assertKubectlCall(t, calls, "logs deepseek-v4-flash-abc --all-containers=true --tail 50")
	assertKubectlCall(t, calls, "logs deepseek-v4-flash-abc --all-containers=true --previous --tail 50")
}

func TestTaskExecutorRetireDeployment(t *testing.T) {
	deleter := &fakeDeleter{result: DeleteResult{Deleted: true, Message: "deleted"}}
	executor := NewTaskExecutorWithKubernetes(&fakeDryRunner{}, &fakeApplier{}, &fakeWatcher{}, deleter)

	result, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypeRetireDeployment,
		Payload: map[string]any{
			"resourceName": "deepseek-v4-flash",
			"namespace":    "dynamo-system",
		},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !deleter.called || result["mode"] != "retire" || result["deleted"] != true {
		t.Fatalf("unexpected retire result: %+v deleter=%+v", result, deleter)
	}
}

func TestTaskExecutorPreviewRequiresManifests(t *testing.T) {
	executor := NewTaskExecutor(&fakeDryRunner{})
	_, err := executor.Execute(context.Background(), management.Task{Type: management.TaskTypePreviewDeploymentDiff})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTaskExecutorNoopForOtherTasks(t *testing.T) {
	executor := NewTaskExecutor(&fakeDryRunner{})
	result, err := executor.Execute(context.Background(), management.Task{Type: management.TaskTypeInspectStatus})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result["mode"] != "noop" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTaskExecutorPropagatesDryRunError(t *testing.T) {
	executor := NewTaskExecutor(&fakeDryRunner{err: errors.New("dry-run failed")})
	_, err := executor.Execute(context.Background(), management.Task{
		Type: management.TaskTypePreviewDeploymentDiff,
		Payload: map[string]any{
			"manifests": []any{map[string]any{"content": "kind: ConfigMap\n"}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertKubectlCall(t *testing.T, calls []string, expected string) {
	t.Helper()
	for _, call := range calls {
		if strings.Contains(call, expected) {
			return
		}
	}
	t.Fatalf("expected kubectl call containing %q, got %+v", expected, calls)
}

type fakeDryRunner struct {
	manifests []Manifest
	result    DryRunResult
	err       error
}

func (r *fakeDryRunner) ServerSideDryRun(_ context.Context, manifests []Manifest) (DryRunResult, error) {
	r.manifests = manifests
	if r.err != nil {
		return DryRunResult{}, r.err
	}
	if r.result == (DryRunResult{}) {
		return DryRunResult{Stdout: "ok"}, nil
	}
	return r.result, nil
}

type fakeApplier struct {
	manifests []Manifest
	result    ApplyResult
	err       error
}

func (a *fakeApplier) Apply(_ context.Context, manifests []Manifest) (ApplyResult, error) {
	a.manifests = manifests
	if a.err != nil {
		return ApplyResult{}, a.err
	}
	if a.result == (ApplyResult{}) {
		return ApplyResult{Stdout: "applied"}, nil
	}
	return a.result, nil
}

type fakeWatcher struct {
	ref    ResourceRef
	result WatchResult
	err    error
}

type fakeDeleter struct {
	ref    ResourceRef
	called bool
	result DeleteResult
	err    error
}

type fakeDiagnosticsCollector struct {
	ref    ResourceRef
	result DiagnosticsResult
	err    error
}

func (w *fakeWatcher) Wait(_ context.Context, ref ResourceRef) (WatchResult, error) {
	w.ref = ref
	if w.err != nil {
		return WatchResult{}, w.err
	}
	if w.result == (WatchResult{}) {
		return WatchResult{Phase: "Ready"}, nil
	}
	return w.result, nil
}

func (d *fakeDeleter) DeleteAndWait(_ context.Context, ref ResourceRef) (DeleteResult, error) {
	d.ref = ref
	d.called = true
	if d.err != nil {
		return DeleteResult{}, d.err
	}
	if d.result == (DeleteResult{}) {
		return DeleteResult{Deleted: true}, nil
	}
	return d.result, nil
}

func (c *fakeDiagnosticsCollector) Fetch(_ context.Context, ref ResourceRef) (DiagnosticsResult, error) {
	c.ref = ref
	if c.err != nil {
		return DiagnosticsResult{}, c.err
	}
	return c.result, nil
}
