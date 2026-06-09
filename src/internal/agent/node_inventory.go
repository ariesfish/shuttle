package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"zhiliu/internal/management"
)

const nodeInventorySchemaVersion = "accelerator-inventory/v1alpha1"

type KubectlNodeInventoryReporter struct {
	Timeout    time.Duration
	Now        func() time.Time
	runKubectl func(ctx context.Context, args ...string) ([]byte, error)
}

type kubectlNodeList struct {
	Items []kubectlNode `json:"items"`
}

type kubectlNode struct {
	Metadata kubectlNodeMetadata `json:"metadata"`
	Spec     kubectlNodeSpec     `json:"spec"`
	Status   kubectlNodeStatus   `json:"status"`
}

type kubectlNodeMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type kubectlNodeSpec struct {
	Taints []kubectlNodeTaint `json:"taints"`
}

type kubectlNodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

type kubectlNodeStatus struct {
	Capacity    map[string]any `json:"capacity"`
	Allocatable map[string]any `json:"allocatable"`
}

type kubectlPodList struct {
	Items []struct{} `json:"items"`
}

func (r KubectlNodeInventoryReporter) Report(ctx context.Context, _ string, agent management.ClusterAgent) (management.ReportAcceleratorInventoryRequest, bool, error) {
	observedAt := nodeInventoryNow(r.Now)
	request := management.ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: nodeInventorySchemaVersion,
		ObservedAt:    observedAt,
		CollectionMetadata: map[string]string{
			"mode": "kubectl-node-inventory",
		},
	}

	runCtx := ctx
	cancel := func() {}
	if r.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, r.Timeout)
	}
	defer cancel()

	runKubectl := r.runKubectl
	if runKubectl == nil {
		runKubectl = defaultInventoryKubectlRunner
	}
	output, err := runKubectl(runCtx, "get", "nodes", "-o", "json")
	if err != nil {
		request.ProbeStatuses = []management.AcceleratorInventoryProbe{nodeInventoryWarning(err)}
		return request, true, nil
	}

	nodes, err := parseKubectlNodeInventory(output, observedAt)
	if err != nil {
		request.ProbeStatuses = []management.AcceleratorInventoryProbe{{Name: "kubernetes-nodes", Status: "warning", Message: boundedInventoryMessage("parse node inventory: " + err.Error())}}
		return request, true, nil
	}
	request.Nodes = nodes
	request.ProbeStatuses = []management.AcceleratorInventoryProbe{{Name: "kubernetes-nodes", Status: "ok", Message: fmt.Sprintf("collected %d node(s)", len(nodes))}}
	request.ProbeStatuses = append(request.ProbeStatuses, dcgmExporterProbeStatus(runCtx, runKubectl))
	return request, true, nil
}

func defaultInventoryKubectlRunner(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "kubectl", args...)
	return command.CombinedOutput()
}

func parseKubectlNodeInventory(contents []byte, observedAt time.Time) ([]management.AcceleratorInventoryNode, error) {
	var nodeList kubectlNodeList
	if err := json.Unmarshal(contents, &nodeList); err != nil {
		return nil, err
	}
	nodes := make([]management.AcceleratorInventoryNode, 0, len(nodeList.Items))
	for _, item := range nodeList.Items {
		name := strings.TrimSpace(item.Metadata.Name)
		if name == "" {
			continue
		}
		capacity := stringifyResourceMap(item.Status.Capacity)
		allocatable := stringifyResourceMap(item.Status.Allocatable)
		labels := cloneInventoryStringMap(item.Metadata.Labels)
		nodes = append(nodes, management.AcceleratorInventoryNode{
			Name:                     name,
			Labels:                   labels,
			Taints:                   formatNodeTaints(item.Spec.Taints),
			Capacity:                 capacity,
			Allocatable:              allocatable,
			AcceleratorResourceNames: acceleratorResourceNames(capacity, allocatable),
			Accelerators:             nvidiaAcceleratorsFromNode(labels, capacity, allocatable),
			ObservedAt:               observedAt,
		})
	}
	return nodes, nil
}

func stringifyResourceMap(input map[string]any) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch typed := value.(type) {
		case string:
			output[key] = typed
		case float64:
			output[key] = fmt.Sprintf("%g", typed)
		default:
			output[key] = fmt.Sprint(typed)
		}
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func formatNodeTaints(input []kubectlNodeTaint) []string {
	if len(input) == 0 {
		return nil
	}
	output := make([]string, 0, len(input))
	for _, taint := range input {
		key := strings.TrimSpace(taint.Key)
		if key == "" {
			continue
		}
		entry := key
		if strings.TrimSpace(taint.Value) != "" {
			entry += "=" + strings.TrimSpace(taint.Value)
		}
		if strings.TrimSpace(taint.Effect) != "" {
			entry += ":" + strings.TrimSpace(taint.Effect)
		}
		output = append(output, entry)
	}
	sort.Strings(output)
	return output
}

func acceleratorResourceNames(capacity map[string]string, allocatable map[string]string) []string {
	resources := map[string]struct{}{}
	for key := range capacity {
		if isAcceleratorResourceName(key) {
			resources[key] = struct{}{}
		}
	}
	for key := range allocatable {
		if isAcceleratorResourceName(key) {
			resources[key] = struct{}{}
		}
	}
	output := make([]string, 0, len(resources))
	for key := range resources {
		output = append(output, key)
	}
	sort.Strings(output)
	return output
}

func isAcceleratorResourceName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || !strings.Contains(name, "/") {
		return false
	}
	if strings.HasPrefix(name, "nvidia.com/") || strings.HasPrefix(name, "amd.com/") || strings.HasPrefix(name, "habana.ai/") || strings.HasPrefix(name, "intel.com/") {
		return true
	}
	return strings.Contains(name, "gpu") || strings.Contains(name, "accelerator") || strings.Contains(name, "mig-")
}

func nvidiaAcceleratorsFromNode(labels map[string]string, capacity map[string]string, allocatable map[string]string) []management.AcceleratorInventoryAccelerator {
	count := resourceQuantityInt(firstNonEmpty(capacity["nvidia.com/gpu"], allocatable["nvidia.com/gpu"], labels["nvidia.com/gpu.count"]))
	product := firstNonEmpty(labels["nvidia.com/gpu.product"], labels["nvidia.com/gpu.name"])
	memoryMiB := resourceQuantityInt(firstNonEmpty(labels["nvidia.com/gpu.memory"], labels["nvidia.com/gpu.memory-mib"], labels["nvidia.com/gpu.mem"]))
	if count == 0 && product == "" && memoryMiB == 0 {
		return nil
	}
	details := map[string]string{}
	for _, key := range []string{
		"nvidia.com/cuda.driver.major",
		"nvidia.com/cuda.driver.minor",
		"nvidia.com/cuda.runtime.major",
		"nvidia.com/cuda.runtime.minor",
		"nvidia.com/gpu.compute.major",
		"nvidia.com/gpu.compute.minor",
	} {
		if value := strings.TrimSpace(labels[key]); value != "" {
			details[key] = value
		}
	}
	if driver := joinVersion(labels["nvidia.com/cuda.driver.major"], labels["nvidia.com/cuda.driver.minor"]); driver != "" {
		details["driverVersion"] = driver
	}
	if cuda := joinVersion(labels["nvidia.com/cuda.runtime.major"], labels["nvidia.com/cuda.runtime.minor"]); cuda != "" {
		details["cudaRuntimeVersion"] = cuda
	}
	if len(details) == 0 {
		details = nil
	}
	return []management.AcceleratorInventoryAccelerator{{
		Vendor:        "nvidia",
		Product:       product,
		DeviceCount:   count,
		MemoryMiB:     memoryMiB,
		VendorDetails: details,
	}}
}

func dcgmExporterProbeStatus(ctx context.Context, runKubectl func(context.Context, ...string) ([]byte, error)) management.AcceleratorInventoryProbe {
	output, err := runKubectl(ctx, "get", "pods", "-A", "-l", "app=nvidia-dcgm-exporter", "-o", "json")
	if err != nil {
		return management.AcceleratorInventoryProbe{Name: "nvidia-dcgm", Status: "warning", Message: boundedInventoryMessage(err.Error())}
	}
	var pods kubectlPodList
	if err := json.Unmarshal(output, &pods); err != nil {
		return management.AcceleratorInventoryProbe{Name: "nvidia-dcgm", Status: "warning", Message: boundedInventoryMessage("parse dcgm exporter pods: " + err.Error())}
	}
	if len(pods.Items) == 0 {
		return management.AcceleratorInventoryProbe{Name: "nvidia-dcgm", Status: "warning", Message: "dcgm exporter not observed"}
	}
	return management.AcceleratorInventoryProbe{Name: "nvidia-dcgm", Status: "ok", Message: fmt.Sprintf("observed %d dcgm exporter pod(s)", len(pods.Items))}
}

func nodeInventoryWarning(err error) management.AcceleratorInventoryProbe {
	message := err.Error()
	if errors.Is(err, context.DeadlineExceeded) {
		message = "kubectl node inventory timed out"
	}
	return management.AcceleratorInventoryProbe{Name: "kubernetes-nodes", Status: "warning", Message: boundedInventoryMessage(message)}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func joinVersion(major string, minor string) string {
	major = strings.TrimSpace(major)
	minor = strings.TrimSpace(minor)
	if major == "" {
		return ""
	}
	if minor == "" {
		return major
	}
	return major + "." + minor
}

func resourceQuantityInt(value string) int {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "MiB")
	value = strings.TrimSuffix(value, "Mi")
	if value == "" {
		return 0
	}
	var output int
	for _, char := range value {
		if char < '0' || char > '9' {
			break
		}
		output = output*10 + int(char-'0')
	}
	return output
}

func boundedInventoryMessage(message string) string {
	message = strings.TrimSpace(strings.ReplaceAll(message, "\x00", ""))
	const maxMessageLength = 300
	if len(message) <= maxMessageLength {
		return message
	}
	return message[:maxMessageLength] + "..."
}

func nodeInventoryNow(now func() time.Time) time.Time {
	if now == nil {
		now = time.Now
	}
	return now().UTC()
}

func cloneInventoryStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
