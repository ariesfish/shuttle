package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"zhiliu/internal/management"
)

func TestKubectlNodeInventoryReporterMapsNodeAndNVIDIAFacts(t *testing.T) {
	observedAt := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	reporter := KubectlNodeInventoryReporter{
		Now: func() time.Time { return observedAt },
		runKubectl: func(_ context.Context, args ...string) ([]byte, error) {
			switch joined := strings.Join(args, " "); joined {
			case "get nodes -o json":
				return []byte(`{
					"items": [
						{
							"metadata": {"name": "node-a", "labels": {"node-role.kubernetes.io/gpu": "", "secret-looking": "not-a-secret-read", "nvidia.com/gpu.product": "NVIDIA-H200-SXM", "nvidia.com/gpu.memory": "143360", "nvidia.com/cuda.driver.major": "550", "nvidia.com/cuda.driver.minor": "54", "nvidia.com/cuda.runtime.major": "12", "nvidia.com/cuda.runtime.minor": "4"}},
							"spec": {"taints": [{"key": "nvidia.com/gpu", "value": "present", "effect": "NoSchedule"}]},
							"status": {
								"capacity": {"cpu": "128", "memory": "1Ti", "nvidia.com/gpu": "8"},
								"allocatable": {"cpu": "120", "memory": "900Gi", "nvidia.com/gpu": "7", "example.com/accelerator": "2"}
							}
						}
					]
				}`), nil
			case "get pods -A -l app=nvidia-dcgm-exporter -o json":
				return []byte(`{"items":[{}]}`), nil
			default:
				t.Fatalf("unexpected kubectl args: %s", joined)
				return nil, nil
			}
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
	if len(node.Accelerators) != 1 || node.Accelerators[0].Vendor != "nvidia" || node.Accelerators[0].Product != "NVIDIA-H200-SXM" || node.Accelerators[0].DeviceCount != 8 || node.Accelerators[0].MemoryMiB != 143360 {
		t.Fatalf("unexpected nvidia facts: %+v", node.Accelerators)
	}
	if node.Accelerators[0].VendorDetails["driverVersion"] != "550.54" || node.Accelerators[0].VendorDetails["cudaRuntimeVersion"] != "12.4" {
		t.Fatalf("unexpected nvidia runtime facts: %+v", node.Accelerators[0].VendorDetails)
	}
	if len(request.ProbeStatuses) != 2 || request.ProbeStatuses[0].Name != "kubernetes-nodes" || request.ProbeStatuses[0].Status != "ok" || request.ProbeStatuses[1].Name != "nvidia-dcgm" || request.ProbeStatuses[1].Status != "ok" {
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

func TestNVIDIAAcceleratorFactsCoverRepresentativeModels(t *testing.T) {
	cases := []struct {
		product string
		memory  string
	}{
		{product: "NVIDIA-H200-SXM", memory: "143360"},
		{product: "NVIDIA-H800-SXM", memory: "81920"},
		{product: "NVIDIA-H100-SXM", memory: "81920"},
		{product: "NVIDIA-A100-SXM4-80GB", memory: "81920"},
	}
	for _, tc := range cases {
		accelerators := nvidiaAcceleratorsFromNode(map[string]string{"nvidia.com/gpu.product": tc.product, "nvidia.com/gpu.memory": tc.memory}, map[string]string{"nvidia.com/gpu": "8"}, nil)
		if len(accelerators) != 1 || accelerators[0].Product != tc.product || accelerators[0].DeviceCount != 8 || accelerators[0].MemoryMiB == 0 {
			t.Fatalf("unexpected accelerator facts for %s: %+v", tc.product, accelerators)
		}
	}
}

func TestNVIDIAAcceleratorFactsAllowUnknownAndMissingRuntimeSignals(t *testing.T) {
	accelerators := nvidiaAcceleratorsFromNode(nil, map[string]string{"nvidia.com/gpu": "4"}, nil)
	if len(accelerators) != 1 || accelerators[0].Vendor != "nvidia" || accelerators[0].Product != "" || accelerators[0].DeviceCount != 4 || accelerators[0].VendorDetails != nil {
		t.Fatalf("unexpected unknown model facts: %+v", accelerators)
	}
}

func TestDCGMProbeWarningDoesNotDropNodeInventory(t *testing.T) {
	reporter := KubectlNodeInventoryReporter{runKubectl: func(_ context.Context, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "get nodes -o json":
			return []byte(`{"items":[{"metadata":{"name":"node-a","labels":{"nvidia.com/gpu.product":"NVIDIA-H100-SXM"}},"status":{"capacity":{"nvidia.com/gpu":"8"},"allocatable":{"nvidia.com/gpu":"8"}}}]}`), nil
		case "get pods -A -l app=nvidia-dcgm-exporter -o json":
			return nil, errors.New("pods is forbidden: cannot list dcgm exporter")
		default:
			return nil, errors.New("unexpected command")
		}
	}}
	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.Nodes) != 1 || len(request.Nodes[0].Accelerators) != 1 {
		t.Fatalf("expected node inventory to survive dcgm warning: %+v", request)
	}
	if len(request.ProbeStatuses) != 2 || request.ProbeStatuses[1].Name != "nvidia-dcgm" || request.ProbeStatuses[1].Status != "warning" {
		t.Fatalf("expected dcgm warning, got %+v", request.ProbeStatuses)
	}
}

func TestAcceleratorResourceNamesIgnoresGeneralResources(t *testing.T) {
	resources := acceleratorResourceNames(map[string]string{"cpu": "128", "memory": "1Ti", "pods": "100", "nvidia.com/gpu": "8"}, map[string]string{"example.com/fpga-accelerator": "1", "hugepages-1Gi": "2"})
	if strings.Join(resources, ",") != "example.com/fpga-accelerator,nvidia.com/gpu" {
		t.Fatalf("unexpected resources: %+v", resources)
	}
}
