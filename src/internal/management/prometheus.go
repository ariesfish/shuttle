package management

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PrometheusClient interface {
	Query(context.Context, string, string) (string, error)
}

type HTTPPrometheusClient struct {
	HTTPClient *http.Client
}

func (c HTTPPrometheusClient) Query(ctx context.Context, baseURL string, query string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("prometheus URL is required")
	}
	endpoint, err := url.Parse(baseURL + "/api/v1/query")
	if err != nil {
		return "", err
	}
	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("prometheus query failed: %s", resp.Status)
	}
	var payload prometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Status != "success" {
		if payload.Error != "" {
			return "", fmt.Errorf("prometheus query failed: %s", payload.Error)
		}
		return "", fmt.Errorf("prometheus query failed: %s", payload.Status)
	}
	return payload.Data.value(), nil
}

type prometheusQueryResponse struct {
	Status string              `json:"status"`
	Data   prometheusQueryData `json:"data"`
	Error  string              `json:"error"`
}

type prometheusQueryData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

func (d prometheusQueryData) value() string {
	switch d.ResultType {
	case "vector":
		var vector []struct {
			Value []any `json:"value"`
		}
		if err := json.Unmarshal(d.Result, &vector); err != nil || len(vector) == 0 || len(vector[0].Value) < 2 {
			return ""
		}
		value, _ := vector[0].Value[1].(string)
		return value
	case "scalar", "string":
		var pair []any
		if err := json.Unmarshal(d.Result, &pair); err != nil || len(pair) < 2 {
			return ""
		}
		value, _ := pair[1].(string)
		return value
	default:
		return ""
	}
}
