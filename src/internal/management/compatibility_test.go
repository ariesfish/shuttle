package management

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestServingApplicationValidationRejectsMissingInventory(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	_, err := store.HeartbeatAgent(agent.ID, HeartbeatRequest{LastInventoryFreshness: string(AcceleratorInventoryFreshnessMissing)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	assertInvalidInputContains(t, err, "accelerator inventory missing")
}

func TestServingApplicationValidationRejectsIncompatibleInventory(t *testing.T) {
	cases := []struct {
		name      string
		memoryMiB int
		gpuCount  int
		rdma      bool
		wantError string
	}{
		{name: "insufficient memory", memoryMiB: 40960, gpuCount: 8, rdma: true, wantError: "insufficient NVIDIA memoryMiB"},
		{name: "insufficient gpu count", memoryMiB: 143360, gpuCount: 4, rdma: true, wantError: "insufficient NVIDIA GPU count"},
		{name: "missing rdma", memoryMiB: 143360, gpuCount: 8, rdma: false, wantError: "missing RDMA connectivity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, project, cluster, agent, artifact := compatibilityStore(t)
			reportCompatibilityInventory(t, store, cluster, agent, tc.memoryMiB, tc.gpuCount, tc.rdma)
			_, err := store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
			assertInvalidInputContains(t, err, tc.wantError)
		})
	}
}

func TestServingApplicationValidationRejectsStaleInventory(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	now := time.Date(2026, 6, 9, 11, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now.Add(-10 * time.Minute) }
	reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, true)
	store.now = func() time.Time { return now }
	_, err := store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	assertInvalidInputContains(t, err, "accelerator inventory stale")
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
	_, err = store.CreateServingApplication(request)
	assertInvalidInputContains(t, err, "accelerator pool")
}

func TestServingApplicationRouteValidatesAcceleratorInventory(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	server := NewServer(store, nil).Routes()

	inventory := requestJSON[AcceleratorInventory](t, server, http.MethodPost, "/v1/clusters/"+cluster.ID+"/inventory", ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Nodes: []AcceleratorInventoryNode{{
			Name:         "node-a",
			Labels:       map[string]string{"pool": "h200"},
			Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: 8, MemoryMiB: 143360}},
			Connectivity: []AcceleratorInventoryConnectivity{{Type: "rdma", Present: true, Confidence: "observed"}},
		}},
	}, http.StatusOK)

	app := requestJSON[ServingApplication](t, server, http.MethodPost, "/v1/apps", compatibleRequest(project, cluster, artifact), http.StatusCreated)
	if app.ValidationInventoryRevision != inventory.Revision {
		t.Fatalf("expected route-created app to record inventory revision, app=%+v inventory=%+v", app, inventory)
	}

	badCluster, err := store.CreateCluster(CreateClusterRequest{Name: "partial"})
	if err != nil {
		t.Fatal(err)
	}
	badAgent, err := store.RegisterAgent(RegisterAgentRequest{ClusterID: badCluster.ID, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	requestJSON[AcceleratorInventory](t, server, http.MethodPost, "/v1/clusters/"+badCluster.ID+"/inventory", ReportAcceleratorInventoryRequest{
		AgentID:       badAgent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		Nodes: []AcceleratorInventoryNode{{
			Name:         "node-a",
			Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: 8, MemoryMiB: 143360}},
			Connectivity: []AcceleratorInventoryConnectivity{{Type: "rdma", Present: false, Confidence: "observed"}},
		}},
	}, http.StatusOK)
	badRequest := compatibleRequest(project, badCluster, artifact)
	assertRouteErrorContains(t, server, http.MethodPost, "/v1/apps", badRequest, http.StatusBadRequest, "missing RDMA connectivity")
}

func TestShortPhase2Routes(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	server := NewServer(store, nil).Routes()

	inventory := requestJSON[AcceleratorInventory](t, server, http.MethodPost, "/v1/clusters/"+cluster.ID+"/inventory", ReportAcceleratorInventoryRequest{AgentID: agent.ID, SchemaVersion: "accelerator-inventory/v1alpha1", ObservedAt: time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC), Nodes: []AcceleratorInventoryNode{{Name: "node-a", Labels: map[string]string{"pool": "h200"}, Accelerators: []AcceleratorInventoryAccelerator{{Vendor: "nvidia", Product: "NVIDIA H200 SXM", DeviceCount: 8, MemoryMiB: 143360}}, Connectivity: []AcceleratorInventoryConnectivity{{Type: "rdma", Present: true, Confidence: "observed"}}}}}, http.StatusOK)
	fetched := requestJSON[AcceleratorInventory](t, server, http.MethodGet, "/v1/clusters/"+cluster.ID+"/inventory", nil, http.StatusOK)
	if fetched.Revision != inventory.Revision {
		t.Fatalf("expected inventory route to read reported revision, fetched=%+v inventory=%+v", fetched, inventory)
	}

	pool := requestJSON[AcceleratorPool](t, server, http.MethodPost, "/v1/pools", CreateAcceleratorPoolRequest{ClusterID: cluster.ID, Name: "h200", NodeSelector: map[string]string{"pool": "h200"}}, http.StatusCreated)
	pools := requestJSON[[]AcceleratorPool](t, server, http.MethodGet, "/v1/pools?clusterId="+cluster.ID, nil, http.StatusOK)
	if len(pools) != 1 || pools[0].ID != pool.ID {
		t.Fatalf("expected pools route to read created pool, pools=%+v pool=%+v", pools, pool)
	}

	app := requestJSON[ServingApplication](t, server, http.MethodPost, "/v1/apps", compatibleRequest(project, cluster, artifact), http.StatusCreated)
	tuning := requestJSON[TuningRecord](t, server, http.MethodPost, "/v1/tunings", CreateTuningRecordRequest{ServingApplicationID: app.ID, Reason: "compat"}, http.StatusCreated)
	fetchedTuning := requestJSON[TuningRecord](t, server, http.MethodGet, "/v1/tunings/"+tuning.ID, nil, http.StatusOK)
	if fetchedTuning.ID != tuning.ID {
		t.Fatalf("expected tuning route to read created tuning, got %+v", fetchedTuning)
	}

	requestJSON[ProductionObservabilityEntryPoints](t, server, http.MethodGet, "/v1/apps/"+app.ID+"/observability/links", nil, http.StatusOK)
	requestJSON[[]AuditRecord](t, server, http.MethodGet, "/v1/audit", nil, http.StatusOK)
}

func assertInvalidInputContains(t *testing.T, err error, want string) {
	t.Helper()
	if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected invalid input containing %q, got %v", want, err)
	}
}

func assertRouteErrorContains(t *testing.T, handler http.Handler, method string, path string, body any, expectedStatus int, want string) {
	t.Helper()
	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != expectedStatus || !strings.Contains(recorder.Body.String(), want) {
		t.Fatalf("%s %s status=%d body=%s, want status=%d containing %q", method, path, recorder.Code, recorder.Body.String(), expectedStatus, want)
	}
}
