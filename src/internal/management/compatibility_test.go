package management

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func compatibilityStore(t *testing.T) (*FileStore, Project, InferenceCluster, ClusterAgent, ModelArtifact) {
	t.Helper()
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: cluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.CreateModelArtifact(CreateModelArtifactRequest{Family: "deepseek-v4", Variant: "flash", Revision: "rev1", PVCMountPath: "/models", PVCModelPath: "snapshot", Quantization: "fp8"})
	if err != nil {
		t.Fatal(err)
	}
	return store, project, cluster, agent, artifact
}

func compatibleRequest(project Project, cluster InferenceCluster, artifact ModelArtifact) CreateServingApplicationRequest {
	return CreateServingApplicationRequest{
		ProjectID:    project.ID,
		Name:         "DeepSeek V4 Flash",
		Model:        ModelIntent{Family: artifact.Family, Variant: artifact.Variant, ArtifactID: artifact.ID, Quantization: artifact.Quantization},
		Placement:    PlacementIntent{ClusterID: cluster.ID, Namespace: "dynamo-system"},
		Runtime:      RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-vllm-dgd-disagg"},
		Service:      ServiceIntent{EndpointName: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
		Optimization: OptimizationIntent{Target: "throughput", ProfilingMode: "disabled"},
	}
}

func reportCompatibilityInventory(t *testing.T, store *FileStore, cluster InferenceCluster, agent ClusterAgent, memoryMiB int, gpuCount int, rdma bool) AcceleratorInventory {
	t.Helper()
	connectivity := []AcceleratorInventoryConnectivity{{Type: "rdma", Present: rdma, Confidence: "observed"}}
	inventory, err := store.ReportAcceleratorInventory(cluster.ID, ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC), Nodes: []AcceleratorInventoryNode{{Name: "node-a", Labels: map[string]string{"pool": "h200"}, Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: gpuCount, MemoryMiB: memoryMiB}}, Connectivity: connectivity}}})
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestServingApplicationValidationUsesCompatibleInventory(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	inventory := reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, true)
	app, err := store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if app.ValidationInventoryRevision != inventory.Revision {
		t.Fatalf("expected inventory revision on app, app=%+v inventory=%+v", app, inventory)
	}
}

func TestServingApplicationValidationRejectsMissingStaleAndIncompatibleInventory(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	_, err := store.HeartbeatAgent(agent.ID, HeartbeatRequest{LastInventoryFreshness: string(AcceleratorInventoryFreshnessMissing)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "accelerator inventory missing") {
		t.Fatalf("expected missing inventory error, got %v", err)
	}
	reportCompatibilityInventory(t, store, cluster, agent, 40960, 8, true)
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "insufficient NVIDIA memoryMiB") {
		t.Fatalf("expected memory error, got %v", err)
	}
	reportCompatibilityInventory(t, store, cluster, agent, 143360, 4, true)
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "insufficient NVIDIA GPU count") {
		t.Fatalf("expected gpu count error, got %v", err)
	}
	reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, false)
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "missing RDMA connectivity") {
		t.Fatalf("expected rdma error, got %v", err)
	}
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now.Add(-10 * time.Minute) }
	reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, true)
	store.now = func() time.Time { return now }
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "accelerator inventory stale") {
		t.Fatalf("expected stale error, got %v", err)
	}
}

func TestServingApplicationValidationUsesAcceleratorPoolPlacement(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	pool, err := store.CreateAcceleratorPool(CreateAcceleratorPoolRequest{ClusterID: cluster.ID, Name: "h200", NodeSelector: map[string]string{"pool": "h200"}})
	if err != nil {
		t.Fatal(err)
	}
	reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, true)
	request := compatibleRequest(project, cluster, artifact)
	request.Placement.AcceleratorPoolID = pool.ID
	if _, err := store.CreateServingApplication(request); err != nil {
		t.Fatalf("expected pool placement compatible, got %v", err)
	}
	request.Placement.AcceleratorPoolID = "missing"
	if _, err := store.CreateServingApplication(request); !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "accelerator pool") {
		t.Fatalf("expected missing pool error, got %v", err)
	}
}
