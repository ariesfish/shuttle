package management

import (
	"context"
	"testing"

	platformtask "zhiliu/internal/task"
)

func TestManagementCommandsCreateProjectRecordsAudit(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	commands := NewManagementCommands(store, nil)
	ctx := withActor(context.Background(), Actor{Name: "alice", Role: "admin"})
	project, err := commands.CreateProject(ctx, CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	records, err := store.ListAuditRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Actor != "alice" || records[0].Action != "create_project" || records[0].Resource != project.ID {
		t.Fatalf("unexpected audit records: %+v", records)
	}
}

func TestManagementCommandsRejectsWrongRole(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	commands := NewManagementCommands(store, nil)
	ctx := withActor(context.Background(), Actor{Name: "viewer", Role: "viewer"})
	if _, err := commands.CreateProject(ctx, CreateProjectRequest{Name: "platform"}); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestManagementCommandsCreateTaskRecordsAudit(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "h200-a"})
	if err != nil {
		t.Fatal(err)
	}
	commands := NewManagementCommands(store, nil)
	ctx := withActor(context.Background(), Actor{Name: "operator", Role: "operator"})
	task, err := commands.CreateTask(ctx, CreateTaskRequest{ClusterID: cluster.ID, Type: platformtask.TaskTypeInspectStatus})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	records, err := store.ListAuditRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Action != "create_task" || records[0].Resource != task.ID {
		t.Fatalf("unexpected audit records: %+v", records)
	}
}
