package management

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakePrometheusClient struct {
	queries []string
	values  map[string]string
	errors  map[string]error
}

func (c *fakePrometheusClient) Query(_ context.Context, _ string, query string) (string, error) {
	c.queries = append(c.queries, query)
	if err, ok := c.errors[query]; ok {
		return "", err
	}
	if value, ok := c.values[query]; ok {
		return value, nil
	}
	return "0", nil
}

type fakeObservabilityStore struct {
	entry ObservabilityEntry
	err   error
}

func (s *fakeObservabilityStore) GetObservabilityEntry(string) (ObservabilityEntry, error) {
	if s.err != nil {
		return ObservabilityEntry{}, s.err
	}
	return s.entry, nil
}

func TestServingApplicationObservabilityRecordsPartialQueryFailures(t *testing.T) {
	queryA := PrometheusQuery{Name: "ok", Description: "successful query", Query: "up"}
	queryB := PrometheusQuery{Name: "fail", Description: "failed query", Query: "down"}
	fake := &fakePrometheusClient{values: map[string]string{"up": "1"}, errors: map[string]error{"down": errors.New("prometheus unavailable")}}
	observability := NewServingApplicationObservability(&fakeObservabilityStore{entry: ObservabilityEntry{ServingApplicationID: "app-1", ClusterID: "cluster-1", Namespace: "tenant-a", PrometheusURL: "http://prometheus.example", PrometheusQueries: []PrometheusQuery{queryA, queryB}}}, fake)

	summary, err := observability.Summary(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(summary.Results) != 2 || summary.Results[0].Value != "1" || summary.Results[1].Error != "prometheus unavailable" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
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
	server.observability = NewServingApplicationObservability(store, fake)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/apps/"+app.ID+"/observability/summary", nil)
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
