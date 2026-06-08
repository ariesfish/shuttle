package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"inference-platform/internal/management"
)

type ManagementClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
	actor      string
	role       string
}

func NewManagementClient(baseURL string, httpClient *http.Client) *ManagementClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &ManagementClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		actor:      "cluster-agent",
		role:       "agent",
	}
}

func (c *ManagementClient) WithAuth(token string, actor string, role string) *ManagementClient {
	c.token = strings.TrimSpace(token)
	if strings.TrimSpace(actor) != "" {
		c.actor = strings.TrimSpace(actor)
	}
	if strings.TrimSpace(role) != "" {
		c.role = strings.TrimSpace(role)
	}
	return c
}

func (c *ManagementClient) Register(ctx context.Context, req management.RegisterAgentRequest) (management.ClusterAgent, error) {
	var agent management.ClusterAgent
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/register", req, &agent); err != nil {
		return management.ClusterAgent{}, err
	}
	return agent, nil
}

func (c *ManagementClient) Heartbeat(ctx context.Context, agentID string, req management.HeartbeatRequest) (management.ClusterAgent, error) {
	var agent management.ClusterAgent
	if err := c.doJSON(ctx, http.MethodPost, "/v1/agents/"+agentID+"/heartbeat", req, &agent); err != nil {
		return management.ClusterAgent{}, err
	}
	return agent, nil
}

func (c *ManagementClient) LeaseTask(ctx context.Context, clusterID string, req management.LeaseTaskRequest) (management.Task, bool, error) {
	var task management.Task
	if err := c.doJSON(ctx, http.MethodPost, "/v1/clusters/"+clusterID+"/tasks:lease", req, &task); err != nil {
		var apiErr APIError
		if AsAPIError(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return management.Task{}, false, nil
		}
		return management.Task{}, false, err
	}
	return task, true, nil
}

func (c *ManagementClient) RenewTaskLease(ctx context.Context, taskID string, req management.RenewTaskLeaseRequest) (management.Task, error) {
	var task management.Task
	if err := c.doJSON(ctx, http.MethodPost, "/v1/tasks/"+taskID+"/lease:renew", req, &task); err != nil {
		return management.Task{}, err
	}
	return task, nil
}

func (c *ManagementClient) CompleteTask(ctx context.Context, taskID string, req management.CompleteTaskRequest) (management.Task, error) {
	var task management.Task
	if err := c.doJSON(ctx, http.MethodPost, "/v1/tasks/"+taskID+"/complete", req, &task); err != nil {
		return management.Task{}, err
	}
	return task, nil
}

func (c *ManagementClient) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var body bytes.Buffer
	if requestBody != nil {
		if err := json.NewEncoder(&body).Encode(requestBody); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &body)
	if err != nil {
		return err
	}
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.actor != "" {
		req.Header.Set("X-Actor", c.actor)
	}
	if c.role != "" {
		req.Header.Set("X-Role", c.role)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		message := payload["error"]
		if message == "" {
			message = resp.Status
		}
		return APIError{StatusCode: resp.StatusCode, Message: message}
	}
	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("management api returned %d: %s", e.StatusCode, e.Message)
}

func AsAPIError(err error, target *APIError) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(APIError)
	if !ok {
		return false
	}
	*target = apiErr
	return true
}
