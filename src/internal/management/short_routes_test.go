package management

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShortRoutesCreateArtifactAppPlanAndTask(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a"})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, slog.Default()).Routes()

	artifact := requestJSON[ModelArtifact](t, server, http.MethodPost, "/v1/artifacts", CreateModelArtifactRequest{Family: "deepseek-v4", Variant: "flash", Revision: "rev1", PVCMountPath: "/models", PVCModelPath: "snapshot", Quantization: "fp8"}, http.StatusCreated)
	plans := requestJSON[[]ServingApplicationCreationPlan](t, server, http.MethodGet, "/v1/artifacts/"+artifact.ID+"/app-plans", nil, http.StatusOK)
	if len(plans) == 0 {
		t.Fatalf("expected app plans")
	}
	app := requestJSON[ServingApplication](t, server, http.MethodPost, "/v1/apps", CreateServingApplicationRequest{
		ProjectID: project.ID,
		Name:      "DeepSeek V4 Flash",
		Model:     plans[0].Model,
		Placement: PlacementIntent{ClusterID: cluster.ID, Namespace: plans[0].Defaults.Namespace},
		Runtime:   plans[0].Runtime,
		Service: ServiceIntent{
			EndpointName: "deepseek-v4-flash",
			Protocol:     plans[0].Defaults.Protocol,
			Exposure:     plans[0].Defaults.Exposure,
		},
		Optimization: OptimizationIntent{Target: plans[0].Defaults.OptimizationTarget, ProfilingMode: plans[0].Defaults.ProfilingMode},
	}, http.StatusCreated)
	task := requestJSON[Task](t, server, http.MethodPost, "/v1/apps/"+app.ID+"/tasks/preview", nil, http.StatusCreated)
	if task.ID == "" || task.ClusterID != cluster.ID {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func requestJSON[T any](t *testing.T, handler http.Handler, method string, path string, body any, expectedStatus int) T {
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
	if recorder.Code != expectedStatus {
		t.Fatalf("%s %s status=%d body=%s", method, path, recorder.Code, recorder.Body.String())
	}
	var value T
	if err := json.NewDecoder(recorder.Body).Decode(&value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return value
}
