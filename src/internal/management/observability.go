package management

import (
	"context"
	"time"
)

// ServingApplicationObservability is the deep Module for Serving Application
// observability summaries. Its Interface hides ObservabilityEntry lookup,
// Prometheus query execution, partial failures, and fetch timestamps behind a
// single summary operation.
type ServingApplicationObservability struct {
	store      ObservabilityStore
	prometheus PrometheusClient
	now        func() time.Time
}

func NewServingApplicationObservability(store ObservabilityStore, prometheus PrometheusClient) *ServingApplicationObservability {
	if prometheus == nil {
		prometheus = HTTPPrometheusClient{}
	}
	return &ServingApplicationObservability{store: store, prometheus: prometheus, now: time.Now}
}

func (o *ServingApplicationObservability) Summary(ctx context.Context, appID string) (ObservabilitySummary, error) {
	entry, err := o.store.GetObservabilityEntry(appID)
	if err != nil {
		return ObservabilitySummary{}, err
	}
	return o.summaryFromEntry(ctx, entry), nil
}

func (o *ServingApplicationObservability) summaryFromEntry(ctx context.Context, entry ObservabilityEntry) ObservabilitySummary {
	now := o.nowUTC()
	summary := ObservabilitySummary{
		ServingApplicationID: entry.ServingApplicationID,
		ClusterID:            entry.ClusterID,
		Namespace:            entry.Namespace,
		PrometheusURL:        entry.PrometheusURL,
		Results:              make([]PrometheusQueryResult, 0, len(entry.PrometheusQueries)),
	}
	for _, query := range entry.PrometheusQueries {
		summary.Results = append(summary.Results, o.queryPrometheus(ctx, entry.PrometheusURL, query, now))
	}
	return summary
}

func (o *ServingApplicationObservability) queryPrometheus(ctx context.Context, prometheusURL string, query PrometheusQuery, fetchedAt time.Time) PrometheusQueryResult {
	result := PrometheusQueryResult{Name: query.Name, Description: query.Description, Query: query.Query, FetchedAt: fetchedAt}
	value, err := o.prometheus.Query(ctx, prometheusURL, query.Query)
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Value = value
	}
	return result
}

func (o *ServingApplicationObservability) nowUTC() time.Time {
	if o == nil || o.now == nil {
		return time.Now().UTC()
	}
	return o.now().UTC()
}
