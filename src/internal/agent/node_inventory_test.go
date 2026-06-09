package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"zhiliu/internal/management"
)

func TestKubectlNodeInventoryReporterMapsNodeFacts(t *testing.T) {
	observedAt := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	reporter := KubectlNodeInventoryReporter{
		Now: func() time.Time { return observedAt },
		runKubectl: func(_ context.Context, args ...string) ([]byte, error) {
			joined := strings.Join(args, " ")
			if joined != "get nodes -o json" {
				t.Fatalf("unexpected kubectl args: %s", joined)
			}
			return []byte(`{
				"items": [
					{
						"metadata": {"name": "node-a", "labels": {"node-role.kubernetes.io/gpu": "", "secret-looking": "not-a-secret-read"}},
						"spec": {"taints": [{"key": "nvidia.com/gpu", "value": "present", "effect": "NoSchedule"}]},
						"status": {
							"capacity": {"cpu": "128", "memory": "1Ti", "nvidia.com/gpu": "8"},
							"allocatable": {"cpu": "120", "memory": "900Gi", "nvidia.com/gpu": "7", "example.com/accelerator": "2"}
						}
					}
				]
			}`), nil
		},
	}

	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if request.SchemaVersion != nodeInventorySchemaVersion || request.AgentID != "agent-1" || !request.ObservedAt.Equal(observedAt) {
		t.Fatalf("unexpected request metadata: %+v", request)
	}
	if len(request.Nodes) != 1 {
		t.Fatalf("expected one node, got %+v", request.Nodes)
	}
	node := request.Nodes[0]
	if node.Name != "node-a" || node.Labels["node-role.kubernetes.io/gpu"] != "" || node.Capacity["nvidia.com/gpu"] != "8" || node.Allocatable["nvidia.com/gpu"] != "7" {
		t.Fatalf("unexpected node facts: %+v", node)
	}
	if len(node.Taints) != 1 || node.Taints[0] != "nvidia.com/gpu=present:NoSchedule" {
		t.Fatalf("unexpected taints: %+v", node.Taints)
	}
	if strings.Join(node.AcceleratorResourceNames, ",") != "example.com/accelerator,nvidia.com/gpu" {
		t.Fatalf("unexpected accelerator resources: %+v", node.AcceleratorResourceNames)
	}
	if len(request.ProbeStatuses) != 1 || request.ProbeStatuses[0].Name != "kubernetes-nodes" || request.ProbeStatuses[0].Status != "ok" {
		t.Fatalf("unexpected probe statuses: %+v", request.ProbeStatuses)
	}
}

func TestKubectlNodeInventoryReporterReturnsWarningOnMissingPermission(t *testing.T) {
	reporter := KubectlNodeInventoryReporter{runKubectl: func(context.Context, ...string) ([]byte, error) {
		return []byte("Error from server (Forbidden): nodes is forbidden: User cannot list resource nodes"), errors.New("Error from server (Forbidden): nodes is forbidden: User cannot list resource nodes")
	}}

	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.Nodes) != 0 {
		t.Fatalf("expected no nodes on missing permission, got %+v", request.Nodes)
	}
	if len(request.ProbeStatuses) != 1 || request.ProbeStatuses[0].Status != "warning" || !strings.Contains(request.ProbeStatuses[0].Message, "Forbidden") {
		t.Fatalf("expected bounded warning, got %+v", request.ProbeStatuses)
	}
}

func TestKubectlNodeInventoryReporterReturnsWarningOnTimeout(t *testing.T) {
	reporter := KubectlNodeInventoryReporter{Timeout: time.Nanosecond, runKubectl: func(ctx context.Context, _ ...string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}

	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.ProbeStatuses) != 1 || request.ProbeStatuses[0].Status != "warning" || request.ProbeStatuses[0].Message != "kubectl node inventory timed out" {
		t.Fatalf("unexpected timeout warning: %+v", request.ProbeStatuses)
	}
}

func TestKubectlNodeInventoryReporterBoundsParseWarnings(t *testing.T) {
	reporter := KubectlNodeInventoryReporter{runKubectl: func(context.Context, ...string) ([]byte, error) {
		return []byte(`not-json-with-secret-token-` + strings.Repeat("x", 500)), nil
	}}

	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.ProbeStatuses) != 1 || request.ProbeStatuses[0].Status != "warning" {
		t.Fatalf("expected parse warning, got %+v", request.ProbeStatuses)
	}
	if len(request.ProbeStatuses[0].Message) > 303 {
		t.Fatalf("expected bounded warning, got %d chars", len(request.ProbeStatuses[0].Message))
	}
}

func TestAcceleratorResourceNamesIgnoresGeneralResources(t *testing.T) {
	resources := acceleratorResourceNames(map[string]string{"cpu": "128", "memory": "1Ti", "pods": "100", "nvidia.com/gpu": "8"}, map[string]string{"example.com/fpga-accelerator": "1", "hugepages-1Gi": "2"})
	if strings.Join(resources, ",") != "example.com/fpga-accelerator,nvidia.com/gpu" {
		t.Fatalf("unexpected resources: %+v", resources)
	}
}
