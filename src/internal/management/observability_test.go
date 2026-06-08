package management

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakePrometheusClient struct {
	queries []string
	values  map[string]string
}

func (c *fakePrometheusClient) Query(_ context.Context, _ string, query string) (string, error) {
	c.queries = append(c.queries, query)
	if value, ok := c.values[query]; ok {
		return value, nil
	}
	return "0", nil
}

func TestGetObservabilitySummaryQueriesPrometheus(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "cluster-a", PrometheusURL: "http://prometheus.example"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.CreateModelArtifact(CreateModelArtifactRequest{Family: "deepseek-v4", Variant: "flash", Revision: "rev1", PVCMountPath: "/models", PVCModelPath: "snapshot", Quantization: "fp8"})
	if err != nil {
		t.Fatal(err)
	}
	app, err := store.CreateServingApplication(CreateServingApplicationRequest{
		ProjectID:    project.ID,
		Name:         "DeepSeek V4 Flash",
		Model:        ModelIntent{Family: "deepseek-v4", Variant: "flash", ArtifactID: artifact.ID, Quantization: "fp8"},
		Placement:    PlacementIntent{ClusterID: cluster.ID, Namespace: "tenant-a"},
		Runtime:      RuntimeIntent{Backend: "vllm", Topology: "pd-disagg", Recipe: "deepseek-v4-flash-vllm-dgd-disagg"},
		Service:      ServiceIntent{EndpointName: "deepseek-v4-flash", Protocol: "openai-compatible", Exposure: "cluster-local"},
		Optimization: OptimizationIntent{Target: "throughput", ProfilingMode: "disabled"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakePrometheusClient{values: map[string]string{}}
	server := NewServer(store, slog.Default())
	server.prometheusClient = fake
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/serving-applications/"+app.ID+"/observability/summary", nil)
	server.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	var summary ObservabilitySummary
	if err := json.NewDecoder(strings.NewReader(recorder.Body.String())).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.ServingApplicationID != app.ID || len(summary.Results) != 3 || len(fake.queries) != 3 {
		t.Fatalf("unexpected summary=%+v queries=%+v", summary, fake.queries)
	}
}
