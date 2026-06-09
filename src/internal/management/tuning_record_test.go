package management

import "testing"

func TestTuningRecordCreateListRead(t *testing.T) {
	store, project, cluster, agent, artifact := compatibilityStore(t)
	inventory := reportCompatibilityInventory(t, store, cluster, agent, 143360, 8, true)
	app, err := store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.CreateTuningRecord("operator", CreateTuningRecordRequest{ServingApplicationID: app.ID, BenchmarkSummary: map[string]any{"throughputTokensPerSecond": 1234}, PlannerSettings: map[string]any{"prefillTp": 4}, Recommendations: []string{"keep current topology"}, Reason: "baseline"})
	if err != nil {
		t.Fatalf("create tuning record: %v", err)
	}
	if record.ServingApplicationID != app.ID || record.ClusterID != cluster.ID || record.ModelArtifactID != artifact.ID || record.ServingRecipeID != app.Runtime.Recipe || record.AcceleratorInventoryRevision != inventory.Revision || record.Actor != "operator" {
		t.Fatalf("unexpected tuning record references: %+v", record)
	}
	if record.BenchmarkSummary["throughputTokensPerSecond"] != 1234 || record.PlannerSettings["prefillTp"] != 4 || len(record.Recommendations) != 1 {
		t.Fatalf("unexpected tuning summary: %+v", record)
	}
	records, err := store.ListTuningRecords(app.ID)
	if err != nil || len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("list tuning records: records=%+v err=%v", records, err)
	}
	fetched, err := store.GetTuningRecord(record.ID)
	if err != nil || fetched.ID != record.ID {
		t.Fatalf("get tuning record: record=%+v err=%v", fetched, err)
	}
	audit, err := store.ListAuditRecords()
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) == 0 || audit[len(audit)-1].Action != "create_tuning_record" || audit[len(audit)-1].Metadata["inventoryRevision"] != inventory.Revision {
		t.Fatalf("missing tuning audit: %+v", audit)
	}
}

func TestTuningRecordRequiresInventoryBackedApplication(t *testing.T) {
	store, err := NewFileStore("")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(CreateProjectRequest{Name: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := store.CreateCluster(CreateClusterRequest{Name: "phase-1"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.CreateModelArtifact(CreateModelArtifactRequest{Family: "deepseek-v4", Variant: "flash", Revision: "rev1", PVCMountPath: "/models", PVCModelPath: "snapshot", Quantization: "fp8"})
	if err != nil {
		t.Fatal(err)
	}
	app, err := store.CreateServingApplication(compatibleRequest(project, cluster, artifact))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTuningRecord("operator", CreateTuningRecordRequest{ServingApplicationID: app.ID}); err == nil {
		t.Fatal("expected tuning record to require validation inventory revision")
	}
}
