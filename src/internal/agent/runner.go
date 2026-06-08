package agent

import (
	"context"
	"log/slog"
	"time"

	"inference-platform/internal/management"
)

type Config struct {
	ManagementURL     string
	ClusterID         string
	Version           string
	Capabilities      map[string]string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
}

type Runner struct {
	client   *ManagementClient
	config   Config
	logger   *slog.Logger
	executor Executor
}

func NewRunner(client *ManagementClient, config Config, logger *slog.Logger) *Runner {
	return NewRunnerWithExecutor(client, config, logger, NewTaskExecutor(nil))
}

func NewRunnerWithExecutor(client *ManagementClient, config Config, logger *slog.Logger, executor Executor) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if executor == nil {
		executor = NewTaskExecutor(nil)
	}
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	return &Runner{client: client, config: config, logger: logger, executor: executor}
}

func (r *Runner) Run(ctx context.Context) error {
	agent, err := r.client.Register(ctx, management.RegisterAgentRequest{
		ClusterID:    r.config.ClusterID,
		Version:      r.config.Version,
		Capabilities: r.config.Capabilities,
	})
	if err != nil {
		return err
	}
	r.logger.Info("agent registered", "agent_id", agent.ID, "cluster_id", agent.ClusterID)

	heartbeatTicker := time.NewTicker(r.config.HeartbeatInterval)
	defer heartbeatTicker.Stop()
	pollTicker := time.NewTicker(r.config.PollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeatTicker.C:
			if _, err := r.client.Heartbeat(ctx, agent.ID, management.HeartbeatRequest{
				Version:      r.config.Version,
				Capabilities: r.config.Capabilities,
			}); err != nil {
				r.logger.Error("heartbeat failed", "error", err)
			} else {
				r.logger.Debug("heartbeat sent", "agent_id", agent.ID)
			}
		case <-pollTicker.C:
			r.pollOnce(ctx, agent)
		}
	}
}

func (r *Runner) pollOnce(ctx context.Context, agent management.ClusterAgent) {
	task, ok, err := r.client.LeaseTask(ctx, agent.ClusterID, management.LeaseTaskRequest{AgentID: agent.ID})
	if err != nil {
		r.logger.Error("lease task failed", "error", err)
		return
	}
	if !ok {
		r.logger.Debug("no task available", "cluster_id", agent.ClusterID)
		return
	}

	r.logger.Info("leased task", "task_id", task.ID, "type", task.Type)
	result, taskErr := r.executor.Execute(ctx, task)
	completeReq := management.CompleteTaskRequest{
		AgentID: agent.ID,
		Status:  management.TaskStatusSucceeded,
		Result:  result,
	}
	if taskErr != nil {
		completeReq.Status = management.TaskStatusFailed
		completeReq.Error = taskErr.Error()
	}
	if _, err := r.client.CompleteTask(ctx, task.ID, completeReq); err != nil {
		r.logger.Error("complete task failed", "task_id", task.ID, "error", err)
		return
	}
	r.logger.Info("completed task", "task_id", task.ID, "status", completeReq.Status)
}
