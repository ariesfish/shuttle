package agent

import (
	"context"
	"log/slog"
	"time"

	"zhiliu/internal/management"
)

type Config struct {
	ManagementURL      string
	ClusterID          string
	Version            string
	Capabilities       map[string]string
	PollInterval       time.Duration
	HeartbeatInterval  time.Duration
	LeaseRenewInterval time.Duration
}

type Runner struct {
	client            *ManagementClient
	config            Config
	logger            *slog.Logger
	executor          Executor
	inventoryReporter InventoryReporter
}

func NewRunner(client *ManagementClient, config Config, logger *slog.Logger) *Runner {
	return NewRunnerWithExecutor(client, config, logger, NewTaskExecutor(nil))
}

func NewRunnerWithExecutor(client *ManagementClient, config Config, logger *slog.Logger, executor Executor) *Runner {
	return NewRunnerWithInventory(client, config, logger, executor, NoopInventoryReporter{})
}

func NewRunnerWithInventory(client *ManagementClient, config Config, logger *slog.Logger, executor Executor, inventoryReporter InventoryReporter) *Runner {
	if logger == nil {
		logger = slog.Default()
	}
	if executor == nil {
		executor = NewTaskExecutor(nil)
	}
	if inventoryReporter == nil {
		inventoryReporter = NoopInventoryReporter{}
	}
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.LeaseRenewInterval == 0 {
		config.LeaseRenewInterval = 10 * time.Second
	}
	return &Runner{client: client, config: config, logger: logger, executor: executor, inventoryReporter: inventoryReporter}
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
	agent = r.reportInventoryOnce(ctx, agent)

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
				Version:                 r.config.Version,
				Capabilities:            r.config.Capabilities,
				LastInventoryRevision:   agent.LastInventoryRevision,
				LastInventoryFreshness:  agent.LastInventoryFreshness,
				LastInventoryObservedAt: agent.LastInventoryObservedAt,
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

func (r *Runner) reportInventoryOnce(ctx context.Context, agent management.ClusterAgent) management.ClusterAgent {
	request, ok, err := r.inventoryReporter.Report(ctx, agent.ClusterID, agent)
	if err != nil {
		r.logger.Error("inventory report build failed", "error", err)
		return agent
	}
	if !ok {
		return agent
	}
	inventory, err := r.client.ReportAcceleratorInventory(ctx, agent.ClusterID, request)
	if err != nil {
		r.logger.Error("inventory report failed", "error", err)
		return agent
	}
	agent.LastInventoryRevision = inventory.Revision
	agent.LastInventoryFreshness = string(inventory.Freshness)
	agent.LastInventoryObservedAt = inventory.ObservedAt
	agent.LastInventoryReportedAt = inventory.ReportedAt
	r.logger.Info("inventory reported", "revision", inventory.Revision, "nodes", len(inventory.Nodes))
	return agent
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
	renewCtx, stopRenew := context.WithCancel(ctx)
	doneRenew := make(chan struct{})
	go r.renewLeaseLoop(renewCtx, agent.ID, task.ID, doneRenew)
	result, taskErr := r.executor.Execute(ctx, task)
	stopRenew()
	<-doneRenew
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

func (r *Runner) renewLeaseLoop(ctx context.Context, agentID string, taskID string, done chan<- struct{}) {
	defer close(done)
	interval := r.config.LeaseRenewInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.client.RenewTaskLease(ctx, taskID, management.RenewTaskLeaseRequest{AgentID: agentID}); err != nil {
				r.logger.Error("renew task lease failed", "task_id", taskID, "error", err)
			} else {
				r.logger.Debug("renewed task lease", "task_id", taskID)
			}
		}
	}
}
