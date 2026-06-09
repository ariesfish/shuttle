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
							"metadata": {"name": "node-a", "labels": {"node-role.kubernetes.io/gpu": "", "secret-looking": "not-a-secret-read", "nvidia.com/gpu.product": "NVIDIA-H200-SXM", "nvidia.com/gpu.memory": "143360", "nvidia.com/cuda.driver.major": "550", "nvidia.com/cuda.driver.minor": "54", "nvidia.com/cuda.runtime.major": "12", "nvidia.com/cuda.runtime.minor": "4", "nvidia.com/nvlink.present": "true", "rdma/ib": "true"}},
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
	if len(node.Connectivity) != 2 || node.Connectivity[0].Type != "nvlink" || !node.Connectivity[0].Present || node.Connectivity[1].Type != "rdma" || !node.Connectivity[1].Present {
		t.Fatalf("unexpected connectivity facts: %+v", node.Connectivity)
	}
	if len(request.ProbeStatuses) != 3 || request.ProbeStatuses[0].Name != "kubernetes-nodes" || request.ProbeStatuses[0].Status != "ok" || request.ProbeStatuses[1].Name != "nvidia-dcgm" || request.ProbeStatuses[1].Status != "ok" || request.ProbeStatuses[2].Name != "connectivity" || request.ProbeStatuses[2].Status != "ok" {
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

func TestKubectlNodeInventoryReporterCoversRepresentativeNVIDIAModels(t *testing.T) {
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
		t.Run(tc.product, func(t *testing.T) {
			reporter := kubectlNodeInventoryReporterWithNodes(t, nodeInventoryJSON(tc.product, tc.memory, "8", nil, nil))
			request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
			if err != nil || !ok {
				t.Fatalf("report: ok=%v err=%v", ok, err)
			}
			if len(request.Nodes) != 1 || len(request.Nodes[0].Accelerators) != 1 {
				t.Fatalf("expected one nvidia accelerator, got %+v", request.Nodes)
			}
			accelerator := request.Nodes[0].Accelerators[0]
			if accelerator.Vendor != "nvidia" || accelerator.Product != tc.product || accelerator.DeviceCount != 8 || accelerator.MemoryMiB == 0 {
				t.Fatalf("unexpected accelerator facts: %+v", accelerator)
			}
		})
	}
}

func TestKubectlNodeInventoryReporterAllowsUnknownNVIDIAModelAndMissingRuntimeSignals(t *testing.T) {
	reporter := kubectlNodeInventoryReporterWithNodes(t, nodeInventoryJSON("", "", "4", nil, nil))
	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.Nodes) != 1 || len(request.Nodes[0].Accelerators) != 1 {
		t.Fatalf("expected unknown nvidia accelerator, got %+v", request.Nodes)
	}
	accelerator := request.Nodes[0].Accelerators[0]
	if accelerator.Vendor != "nvidia" || accelerator.Product != "" || accelerator.DeviceCount != 4 || accelerator.VendorDetails != nil {
		t.Fatalf("unexpected unknown model facts: %+v", accelerator)
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
	if len(request.ProbeStatuses) != 3 || request.ProbeStatuses[1].Name != "nvidia-dcgm" || request.ProbeStatuses[1].Status != "warning" {
		t.Fatalf("expected dcgm warning, got %+v", request.ProbeStatuses)
	}
}

func TestKubectlNodeInventoryReporterRepresentsUnavailableAndIncompleteConnectivitySignals(t *testing.T) {
	reporter := kubectlNodeInventoryReporterWithNodes(t, nodeInventoryJSON("NVIDIA-H200-SXM", "143360", "8", map[string]string{"nvidia.com/nvlink.present": "false", "feature.node.kubernetes.io/network-sriov.capable": ""}, nil))
	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.Nodes) != 1 || len(request.Nodes[0].Connectivity) != 2 {
		t.Fatalf("expected connectivity facts, got %+v", request.Nodes)
	}
	facts := request.Nodes[0].Connectivity
	if facts[0].Type != "nvlink" || facts[0].Present || facts[0].Confidence != "observed" || facts[1].Type != "rdma" || !facts[1].Present || facts[1].Confidence != "incomplete" {
		t.Fatalf("unexpected connectivity facts: %+v", facts)
	}
	if len(request.ProbeStatuses) != 3 || request.ProbeStatuses[2].Name != "connectivity" || request.ProbeStatuses[2].Status != "warning" || !strings.Contains(request.ProbeStatuses[2].Message, "nvlink") {
		t.Fatalf("unexpected connectivity probe: %+v", request.ProbeStatuses)
	}
}

func TestKubectlNodeInventoryReporterReportsOnlyAcceleratorResourceNames(t *testing.T) {
	reporter := kubectlNodeInventoryReporterWithNodes(t, `{
		"items": [{
			"metadata": {"name": "node-a", "labels": {"nvidia.com/gpu.product": "NVIDIA-H200-SXM"}},
			"status": {
				"capacity": {"cpu": "128", "memory": "1Ti", "pods": "100", "nvidia.com/gpu": "8"},
				"allocatable": {"example.com/fpga-accelerator": "1", "hugepages-1Gi": "2"}
			}
		}]
	}`)
	request, ok, err := reporter.Report(context.Background(), "cluster-1", management.ClusterAgent{ID: "agent-1", ClusterID: "cluster-1"})
	if err != nil || !ok {
		t.Fatalf("report: ok=%v err=%v", ok, err)
	}
	if len(request.Nodes) != 1 || strings.Join(request.Nodes[0].AcceleratorResourceNames, ",") != "example.com/fpga-accelerator,nvidia.com/gpu" {
		t.Fatalf("unexpected resources: %+v", request.Nodes)
	}
}

func kubectlNodeInventoryReporterWithNodes(t *testing.T, nodesJSON string) KubectlNodeInventoryReporter {
	t.Helper()
	return KubectlNodeInventoryReporter{runKubectl: func(_ context.Context, args ...string) ([]byte, error) {
		switch joined := strings.Join(args, " "); joined {
		case "get nodes -o json":
			return []byte(nodesJSON), nil
		case "get pods -A -l app=nvidia-dcgm-exporter -o json":
			return []byte(`{"items":[{}]}`), nil
		default:
			t.Fatalf("unexpected kubectl args: %s", joined)
			return nil, nil
		}
	}}
}

func nodeInventoryJSON(product string, memory string, gpuCount string, labels map[string]string, annotations map[string]string) string {
	mergedLabels := map[string]string{}
	if product != "" {
		mergedLabels["nvidia.com/gpu.product"] = product
	}
	if memory != "" {
		mergedLabels["nvidia.com/gpu.memory"] = memory
	}
	for key, value := range labels {
		mergedLabels[key] = value
	}
	return nodeInventoryJSONWithMaps(mergedLabels, annotations, map[string]string{"nvidia.com/gpu": gpuCount}, nil)
}

func nodeInventoryJSONWithMaps(labels map[string]string, annotations map[string]string, capacity map[string]string, allocatable map[string]string) string {
	labelsJSON := inventoryStringMapJSON(labels)
	annotationsJSON := inventoryStringMapJSON(annotations)
	capacityJSON := inventoryStringMapJSON(capacity)
	allocatableJSON := inventoryStringMapJSON(allocatable)
	return `{"items":[{"metadata":{"name":"node-a","labels":` + labelsJSON + `,"annotations":` + annotationsJSON + `},"status":{"capacity":` + capacityJSON + `,"allocatable":` + allocatableJSON + `}}]}`
}

func inventoryStringMapJSON(values map[string]string) string {
	if len(values) == 0 {
		return `{}`
	}
	entries := make([]string, 0, len(values))
	for key, value := range values {
		entries = append(entries, `"`+key+`":"`+value+`"`)
	}
	return `{` + strings.Join(entries, ",") + `}`
}
