package management

import (
	"strings"

	platformtask "zhiliu/internal/task"
)

type RenderedDeploymentManifest struct {
	Name    string
	Content string
}

type EndpointOperation string

const (
	EndpointOperationNone   EndpointOperation = ""
	EndpointOperationUpsert EndpointOperation = "upsert"
	EndpointOperationRemove EndpointOperation = "remove"
)

func servingApplicationIDForTask(tasks platformtask.Registry, task Task) (string, error) {
	payload, err := tasks.DecodePayload(platformtask.NewDTO(task.ID, task.ClusterID, task.Type, task.Payload, nil, ""))
	if err != nil {
		return "", err
	}
	return payload.ServingApplicationID(), nil
}

func taskFailureReason(task Task) string {
	if strings.TrimSpace(task.Error) != "" {
		return task.Error
	}
	return taskResultMessage(platformtask.DefaultRegistry(), task)
}

func taskResultMessage(tasks platformtask.Registry, task Task) string {
	result, err := tasks.DecodeResult(platformtask.NewDTO(task.ID, task.ClusterID, task.Type, task.Payload, task.Result, task.Error))
	if err == nil {
		switch value := result.(type) {
		case platformtask.DeploymentResult:
			if strings.TrimSpace(value.Message) != "" {
				return strings.TrimSpace(value.Message)
			}
			if strings.TrimSpace(value.Phase) != "" {
				return "task result phase: " + strings.TrimSpace(value.Phase)
			}
		case platformtask.RetireResult:
			if strings.TrimSpace(value.Message) != "" {
				return strings.TrimSpace(value.Message)
			}
		}
	}
	return string(task.Type) + " completed"
}
