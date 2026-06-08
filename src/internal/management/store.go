package management

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrTaskLeaseHeld = errors.New("task lease held by another agent")
)

type Store interface {
	CreateProject(CreateProjectRequest) (Project, error)
	ListProjects() ([]Project, error)
	CreateCluster(CreateClusterRequest) (InferenceCluster, error)
	ListClusters() ([]InferenceCluster, error)
	GetCluster(string) (InferenceCluster, error)
	RegisterAgent(RegisterAgentRequest) (ClusterAgent, error)
	HeartbeatAgent(string, HeartbeatRequest) (ClusterAgent, error)
	ListAgents() ([]ClusterAgent, error)
	CreateModelArtifact(CreateModelArtifactRequest) (ModelArtifact, error)
	ListModelArtifacts() ([]ModelArtifact, error)
	GetModelArtifact(string) (ModelArtifact, error)
	CreateServingApplication(CreateServingApplicationRequest) (ServingApplication, error)
	ListServingApplications() ([]ServingApplication, error)
	GetServingApplication(string) (ServingApplication, error)
	ListServingApplicationTransitions(string) ([]ServingApplicationTransition, error)
	ListEndpoints() ([]EndpointRegistryEntry, error)
	GetObservabilityEntry(string) (ObservabilityEntry, error)
	ListAuditRecords() ([]AuditRecord, error)
	RecordAudit(actor, action, resource string, metadata map[string]any) (AuditRecord, error)
	CreateTask(CreateTaskRequest) (Task, error)
	CreatePreviewTask(CreatePreviewTaskRequest) (Task, error)
	CreateApplyTask(CreateApplyTaskRequest) (Task, error)
	CreateRedeployTask(CreateRedeployTaskRequest) (Task, error)
	CreateRetireTask(CreateRetireTaskRequest) (Task, error)
	CreateDiagnosticsTask(CreateDiagnosticsTaskRequest) (Task, error)
	ListTasks(clusterID string) ([]Task, error)
	LeaseNextTask(clusterID string, req LeaseTaskRequest, ttl time.Duration) (Task, error)
	RenewTaskLease(taskID string, req RenewTaskLeaseRequest, ttl time.Duration) (Task, error)
	CompleteTask(taskID string, req CompleteTaskRequest) (Task, error)
}

type FileStore struct {
	mu      sync.Mutex
	path    string
	data    storeData
	now     func() time.Time
	persist func(storeData) error
}

type storeData struct {
	NextID              int                                     `json:"nextId"`
	Projects            map[string]Project                      `json:"projects"`
	Clusters            map[string]InferenceCluster             `json:"clusters"`
	Agents              map[string]ClusterAgent                 `json:"agents"`
	ModelArtifacts      map[string]ModelArtifact                `json:"modelArtifacts"`
	ServingApplications map[string]ServingApplication           `json:"servingApplications"`
	Transitions         map[string]ServingApplicationTransition `json:"transitions"`
	Endpoints           map[string]EndpointRegistryEntry        `json:"endpoints"`
	AuditRecords        map[string]AuditRecord                  `json:"auditRecords"`
	Tasks               map[string]Task                         `json:"tasks"`
}

func NewFileStore(path string) (*FileStore, error) {
	store := &FileStore{path: path, now: time.Now}
	store.persist = store.persistFile
	store.data = newStoreData()
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func newStoreData() storeData {
	return storeData{
		NextID:              1,
		Projects:            map[string]Project{},
		Clusters:            map[string]InferenceCluster{},
		Agents:              map[string]ClusterAgent{},
		ModelArtifacts:      map[string]ModelArtifact{},
		ServingApplications: map[string]ServingApplication{},
		Transitions:         map[string]ServingApplicationTransition{},
		Endpoints:           map[string]EndpointRegistryEntry{},
		AuditRecords:        map[string]AuditRecord{},
		Tasks:               map[string]Task{},
	}
}

func (s *FileStore) CreateProject(req CreateProjectRequest) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Project{}, fmt.Errorf("%w: project name is required", ErrInvalidInput)
	}

	now := s.now().UTC()
	project := Project{ID: s.nextID("project"), Name: name, CreatedAt: now, UpdatedAt: now}
	s.data.Projects[project.ID] = project
	return project, s.saveLocked()
}

func (s *FileStore) ListProjects() ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projects := make([]Project, 0, len(s.data.Projects))
	for _, project := range s.data.Projects {
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].CreatedAt.Before(projects[j].CreatedAt) })
	return projects, nil
}

func (s *FileStore) CreateCluster(req CreateClusterRequest) (InferenceCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return InferenceCluster{}, fmt.Errorf("%w: cluster name is required", ErrInvalidInput)
	}

	now := s.now().UTC()
	cluster := InferenceCluster{
		ID:            s.nextID("cluster"),
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		PrometheusURL: strings.TrimSpace(req.PrometheusURL),
		GrafanaURL:    strings.TrimSpace(req.GrafanaURL),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.data.Clusters[cluster.ID] = cluster
	return cluster, s.saveLocked()
}

func (s *FileStore) ListClusters() ([]InferenceCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusters := make([]InferenceCluster, 0, len(s.data.Clusters))
	for _, cluster := range s.data.Clusters {
		clusters = append(clusters, cluster)
	}
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].CreatedAt.Before(clusters[j].CreatedAt) })
	return clusters, nil
}

func (s *FileStore) GetCluster(id string) (InferenceCluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cluster, ok := s.data.Clusters[id]
	if !ok {
		return InferenceCluster{}, ErrNotFound
	}
	return cluster, nil
}

func (s *FileStore) RegisterAgent(req RegisterAgentRequest) (ClusterAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Clusters[req.ClusterID]; !ok {
		return ClusterAgent{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}

	now := s.now().UTC()
	for _, existing := range s.data.Agents {
		if existing.ClusterID == req.ClusterID {
			existing.Version = strings.TrimSpace(req.Version)
			existing.Capabilities = cloneStringMap(req.Capabilities)
			existing.LastHeartbeat = now
			existing.UpdatedAt = now
			s.data.Agents[existing.ID] = existing
			return existing, s.saveLocked()
		}
	}

	agent := ClusterAgent{
		ID:            s.nextID("agent"),
		ClusterID:     req.ClusterID,
		Version:       strings.TrimSpace(req.Version),
		Capabilities:  cloneStringMap(req.Capabilities),
		LastHeartbeat: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.data.Agents[agent.ID] = agent
	return agent, s.saveLocked()
}

func (s *FileStore) HeartbeatAgent(id string, req HeartbeatRequest) (ClusterAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.data.Agents[id]
	if !ok {
		return ClusterAgent{}, ErrNotFound
	}

	now := s.now().UTC()
	if strings.TrimSpace(req.Version) != "" {
		agent.Version = strings.TrimSpace(req.Version)
	}
	if req.Capabilities != nil {
		agent.Capabilities = cloneStringMap(req.Capabilities)
	}
	agent.LastHeartbeat = now
	agent.UpdatedAt = now
	s.data.Agents[id] = agent
	return agent, s.saveLocked()
}

func (s *FileStore) ListAgents() ([]ClusterAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agents := make([]ClusterAgent, 0, len(s.data.Agents))
	for _, agent := range s.data.Agents {
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].CreatedAt.Before(agents[j].CreatedAt) })
	return agents, nil
}

func (s *FileStore) CreateModelArtifact(req CreateModelArtifactRequest) (ModelArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	family := strings.TrimSpace(req.Family)
	variant := strings.TrimSpace(req.Variant)
	revision := strings.TrimSpace(req.Revision)
	mountPath := strings.TrimSpace(req.PVCMountPath)
	modelPath := strings.TrimSpace(req.PVCModelPath)
	quantization := strings.TrimSpace(req.Quantization)
	if family == "" || variant == "" || revision == "" || mountPath == "" || modelPath == "" || quantization == "" {
		return ModelArtifact{}, fmt.Errorf("%w: family, variant, revision, pvcMountPath, pvcModelPath, and quantization are required", ErrInvalidInput)
	}

	now := s.now().UTC()
	artifact := ModelArtifact{
		ID:            s.nextID("artifact"),
		Family:        family,
		Variant:       variant,
		Revision:      revision,
		PVCName:       strings.TrimSpace(req.PVCName),
		PVCMountPath:  mountPath,
		PVCModelPath:  modelPath,
		HostCachePath: strings.TrimSpace(req.HostCachePath),
		Quantization:  quantization,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.data.ModelArtifacts[artifact.ID] = artifact
	return artifact, s.saveLocked()
}

func (s *FileStore) ListModelArtifacts() ([]ModelArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	artifacts := make([]ModelArtifact, 0, len(s.data.ModelArtifacts))
	for _, artifact := range s.data.ModelArtifacts {
		artifacts = append(artifacts, artifact)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt) })
	return artifacts, nil
}

func (s *FileStore) GetModelArtifact(id string) (ModelArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	artifact, ok := s.data.ModelArtifacts[id]
	if !ok {
		return ModelArtifact{}, ErrNotFound
	}
	return artifact, nil
}

func (s *FileStore) CreateServingApplication(req CreateServingApplicationRequest) (ServingApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" || strings.TrimSpace(req.ProjectID) == "" {
		return ServingApplication{}, fmt.Errorf("%w: name and projectId are required", ErrInvalidInput)
	}
	if _, ok := s.data.Projects[req.ProjectID]; !ok {
		return ServingApplication{}, fmt.Errorf("%w: project does not exist", ErrInvalidInput)
	}
	if _, ok := s.data.Clusters[req.Placement.ClusterID]; !ok {
		return ServingApplication{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}
	artifact, ok := s.data.ModelArtifacts[req.Model.ArtifactID]
	if !ok {
		return ServingApplication{}, fmt.Errorf("%w: model artifact does not exist", ErrInvalidInput)
	}
	if err := validateServingApplicationIntent(req, artifact); err != nil {
		return ServingApplication{}, err
	}

	now := s.now().UTC()
	app := ServingApplication{
		ID:            s.nextID("app"),
		ProjectID:     req.ProjectID,
		Name:          name,
		Model:         req.Model,
		Placement:     req.Placement,
		Runtime:       req.Runtime,
		Service:       req.Service,
		Optimization:  req.Optimization,
		DesiredState:  "Active",
		Phase:         ServingApplicationPhaseDraft,
		ActiveVersion: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.data.ServingApplications[app.ID] = app
	s.recordServingApplicationTransitionLocked(app.ID, "system", "", "", app.Phase, "created")
	return app, s.saveLocked()
}

func (s *FileStore) ListServingApplications() ([]ServingApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	apps := make([]ServingApplication, 0, len(s.data.ServingApplications))
	for _, app := range s.data.ServingApplications {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].CreatedAt.Before(apps[j].CreatedAt) })
	return apps, nil
}

func (s *FileStore) GetServingApplication(id string) (ServingApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data.ServingApplications[id]
	if !ok {
		return ServingApplication{}, ErrNotFound
	}
	return app, nil
}

func (s *FileStore) ListServingApplicationTransitions(appID string) ([]ServingApplicationTransition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.ServingApplications[appID]; !ok {
		return nil, ErrNotFound
	}
	transitions := make([]ServingApplicationTransition, 0)
	for _, transition := range s.data.Transitions {
		if transition.ServingApplicationID == appID {
			transitions = append(transitions, transition)
		}
	}
	sort.Slice(transitions, func(i, j int) bool { return transitions[i].CreatedAt.Before(transitions[j].CreatedAt) })
	return transitions, nil
}

func (s *FileStore) ListEndpoints() ([]EndpointRegistryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpoints := make([]EndpointRegistryEntry, 0, len(s.data.Endpoints))
	for _, endpoint := range s.data.Endpoints {
		endpoints = append(endpoints, endpoint)
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].CreatedAt.Before(endpoints[j].CreatedAt) })
	return endpoints, nil
}

func (s *FileStore) GetObservabilityEntry(appID string) (ObservabilityEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data.ServingApplications[appID]
	if !ok {
		return ObservabilityEntry{}, ErrNotFound
	}
	cluster, ok := s.data.Clusters[app.Placement.ClusterID]
	if !ok {
		return ObservabilityEntry{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}
	return buildObservabilityEntry(app, cluster), nil
}

func (s *FileStore) ListAuditRecords() ([]AuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]AuditRecord, 0, len(s.data.AuditRecords))
	for _, record := range s.data.AuditRecords {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt.Before(records[j].CreatedAt) })
	return records, nil
}

func (s *FileStore) RecordAudit(actor, action, resource string, metadata map[string]any) (AuditRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	record := AuditRecord{
		ID:        s.nextID("audit"),
		Actor:     strings.TrimSpace(actor),
		Action:    strings.TrimSpace(action),
		Resource:  strings.TrimSpace(resource),
		Metadata:  cloneAnyMap(metadata),
		CreatedAt: now,
	}
	s.data.AuditRecords[record.ID] = record
	return record, s.saveLocked()
}

func (s *FileStore) CreateTask(req CreateTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Clusters[req.ClusterID]; !ok {
		return Task{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}
	if !isAllowedTaskType(req.Type) {
		return Task{}, fmt.Errorf("%w: unsupported task type", ErrInvalidInput)
	}

	task := s.newTaskLocked(req)
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) newTaskLocked(req CreateTaskRequest) Task {
	now := s.now().UTC()
	return Task{
		ID:        s.nextID("task"),
		ClusterID: req.ClusterID,
		Type:      req.Type,
		Status:    TaskStatusPending,
		Payload:   cloneAnyMap(req.Payload),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *FileStore) CreatePreviewTask(req CreatePreviewTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, err := s.newRenderedTaskLocked(req.ServingApplicationID, TaskTypePreviewDeploymentDiff)
	if err != nil {
		return Task{}, err
	}
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) CreateApplyTask(req CreateApplyTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, err := s.newRenderedTaskLocked(req.ServingApplicationID, TaskTypeApplyDeployment)
	if err != nil {
		return Task{}, err
	}
	s.setServingApplicationPhaseLocked(req.ServingApplicationID, "system", task.ID, ServingApplicationPhaseApplying, "apply task created")
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) CreateRedeployTask(req CreateRedeployTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, err := s.newRenderedTaskLocked(req.ServingApplicationID, TaskTypeDeleteBeforeApply)
	if err != nil {
		return Task{}, err
	}
	s.setServingApplicationPhaseLocked(req.ServingApplicationID, "system", task.ID, ServingApplicationPhaseApplying, "redeploy task created")
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) CreateRetireTask(req CreateRetireTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data.ServingApplications[req.ServingApplicationID]
	if !ok {
		return Task{}, ErrNotFound
	}
	task := s.newTaskLocked(CreateTaskRequest{
		ClusterID: app.Placement.ClusterID,
		Type:      TaskTypeRetireDeployment,
		Payload: map[string]any{
			"servingApplicationId": app.ID,
			"resourceName":         kubernetesName(app.Name),
			"namespace":            app.Placement.Namespace,
		},
	})
	s.setServingApplicationPhaseLocked(req.ServingApplicationID, "system", task.ID, ServingApplicationPhaseRetiring, "retire task created")
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) CreateDiagnosticsTask(req CreateDiagnosticsTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data.ServingApplications[req.ServingApplicationID]
	if !ok {
		return Task{}, ErrNotFound
	}
	resourceName := kubernetesName(app.Name)
	if resourceName == "" {
		resourceName = kubernetesName(app.ID)
	}
	task := s.newTaskLocked(CreateTaskRequest{
		ClusterID: app.Placement.ClusterID,
		Type:      TaskTypeFetchDiagnostics,
		Payload: map[string]any{
			"servingApplicationId": app.ID,
			"resourceName":         resourceName,
			"namespace":            app.Placement.Namespace,
		},
	})
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) setServingApplicationPhaseLocked(appID string, actor string, taskID string, phase ServingApplicationPhase, reason string) {
	app := s.data.ServingApplications[appID]
	from := app.Phase
	if from == phase {
		return
	}
	app.Phase = phase
	app.UpdatedAt = s.now().UTC()
	s.data.ServingApplications[app.ID] = app
	s.recordServingApplicationTransitionLocked(app.ID, actor, taskID, from, phase, reason)
}

func (s *FileStore) recordServingApplicationTransitionLocked(appID string, actor string, taskID string, from ServingApplicationPhase, to ServingApplicationPhase, reason string) {
	if actor == "" {
		actor = "system"
	}
	now := s.now().UTC()
	transition := ServingApplicationTransition{
		ID:                   s.nextID("transition"),
		ServingApplicationID: appID,
		Actor:                actor,
		TaskID:               strings.TrimSpace(taskID),
		From:                 from,
		To:                   to,
		Reason:               strings.TrimSpace(reason),
		CreatedAt:            now,
	}
	s.data.Transitions[transition.ID] = transition
}

func (s *FileStore) newRenderedTaskLocked(appID string, taskType TaskType) (Task, error) {
	app, ok := s.data.ServingApplications[appID]
	if !ok {
		return Task{}, ErrNotFound
	}
	artifact, ok := s.data.ModelArtifacts[app.Model.ArtifactID]
	if !ok {
		return Task{}, fmt.Errorf("%w: model artifact does not exist", ErrInvalidInput)
	}
	manifest, err := RenderKnownTemplate(app, artifact)
	if err != nil {
		return Task{}, err
	}
	return s.newTaskLocked(CreateTaskRequest{
		ClusterID: app.Placement.ClusterID,
		Type:      taskType,
		Payload: map[string]any{
			"servingApplicationId": app.ID,
			"resourceName":         kubernetesName(app.Name),
			"namespace":            app.Placement.Namespace,
			"endpointName":         app.Service.EndpointName,
			"protocol":             app.Service.Protocol,
			"exposure":             app.Service.Exposure,
			"manifests": []any{map[string]any{
				"name":    manifest.Name,
				"content": manifest.Content,
			}},
		},
	}), nil
}

func (s *FileStore) ListTasks(clusterID string) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := make([]Task, 0, len(s.data.Tasks))
	for _, task := range s.data.Tasks {
		if clusterID == "" || task.ClusterID == clusterID {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	return tasks, nil
}

func (s *FileStore) LeaseNextTask(clusterID string, req LeaseTaskRequest, ttl time.Duration) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.data.Agents[req.AgentID]
	if !ok {
		return Task{}, fmt.Errorf("%w: agent does not exist", ErrInvalidInput)
	}
	if agent.ClusterID != clusterID {
		return Task{}, fmt.Errorf("%w: agent does not belong to cluster", ErrInvalidInput)
	}

	now := s.now().UTC()
	var selected *Task
	for _, task := range s.sortedTasksLocked() {
		if task.ClusterID != clusterID {
			continue
		}
		leaseExpired := task.Status == TaskStatusLeased && !task.LeaseExpiresAt.After(now)
		if task.Status == TaskStatusPending || leaseExpired {
			copy := task
			selected = &copy
			break
		}
	}
	if selected == nil {
		return Task{}, ErrNotFound
	}

	selected.Status = TaskStatusLeased
	selected.LeaseOwner = req.AgentID
	selected.LeaseExpiresAt = now.Add(ttl)
	selected.UpdatedAt = now
	s.data.Tasks[selected.ID] = *selected
	return *selected, s.saveLocked()
}

func (s *FileStore) RenewTaskLease(taskID string, req RenewTaskLeaseRequest, ttl time.Duration) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.data.Tasks[taskID]
	if !ok {
		return Task{}, ErrNotFound
	}
	if task.Status == TaskStatusSucceeded || task.Status == TaskStatusFailed {
		return task, nil
	}
	if task.Status != TaskStatusLeased || task.LeaseOwner != req.AgentID {
		return Task{}, ErrTaskLeaseHeld
	}

	now := s.now().UTC()
	task.LeaseExpiresAt = now.Add(ttl)
	task.UpdatedAt = now
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) CompleteTask(taskID string, req CompleteTaskRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.data.Tasks[taskID]
	if !ok {
		return Task{}, ErrNotFound
	}
	if req.Status != TaskStatusSucceeded && req.Status != TaskStatusFailed {
		return Task{}, fmt.Errorf("%w: completion status must be succeeded or failed", ErrInvalidInput)
	}
	if task.Status == TaskStatusSucceeded || task.Status == TaskStatusFailed {
		if task.LeaseOwner == req.AgentID && task.Status == req.Status {
			return task, nil
		}
		return Task{}, ErrTaskLeaseHeld
	}
	if task.LeaseOwner != req.AgentID {
		return Task{}, ErrTaskLeaseHeld
	}

	now := s.now().UTC()
	task.Status = req.Status
	task.Result = cloneAnyMap(req.Result)
	task.Error = strings.TrimSpace(req.Error)
	task.UpdatedAt = now
	s.data.Tasks[task.ID] = task
	s.updateServingApplicationPhaseForTaskLocked(task)
	return task, s.saveLocked()
}

func (s *FileStore) updateServingApplicationPhaseForTaskLocked(task Task) {
	appID, _ := task.Payload["servingApplicationId"].(string)
	if strings.TrimSpace(appID) == "" {
		return
	}
	app, ok := s.data.ServingApplications[appID]
	if !ok {
		return
	}

	if task.Status == TaskStatusFailed {
		s.setServingApplicationPhaseLocked(app.ID, task.LeaseOwner, task.ID, ServingApplicationPhaseFailed, taskFailureReason(task))
		return
	}
	if task.Status != TaskStatusSucceeded {
		return
	}

	switch task.Type {
	case TaskTypePreviewDeploymentDiff:
		s.setServingApplicationPhaseLocked(app.ID, task.LeaseOwner, task.ID, ServingApplicationPhaseValidated, "preview succeeded")
	case TaskTypeApplyDeployment, TaskTypeDeleteBeforeApply:
		phase := ServingApplicationPhaseReady
		reason := "deployment ready"
		if resultPhase, _ := task.Result["phase"].(string); strings.EqualFold(resultPhase, "failed") || strings.EqualFold(resultPhase, "error") {
			phase = ServingApplicationPhaseFailed
			reason = taskResultMessage(task)
		}
		s.setServingApplicationPhaseLocked(app.ID, task.LeaseOwner, task.ID, phase, reason)
		if phase == ServingApplicationPhaseReady {
			updatedApp := s.data.ServingApplications[app.ID]
			updatedApp = s.upsertEndpointForTaskLocked(updatedApp, task)
			updatedApp.UpdatedAt = s.now().UTC()
			s.data.ServingApplications[updatedApp.ID] = updatedApp
		}
	case TaskTypeRetireDeployment:
		s.setServingApplicationPhaseLocked(app.ID, task.LeaseOwner, task.ID, ServingApplicationPhaseRetired, "retire succeeded")
		s.removeEndpointForServingApplicationLocked(app.ID)
	default:
		return
	}
}

func (s *FileStore) upsertEndpointForTaskLocked(app ServingApplication, task Task) ServingApplication {
	endpointURL, _ := task.Result["endpointUrl"].(string)
	if strings.TrimSpace(endpointURL) == "" {
		endpointURL = defaultEndpointURL(app)
	}
	ready := app.Phase == ServingApplicationPhaseReady
	for _, endpoint := range s.data.Endpoints {
		if endpoint.ServingApplicationID == app.ID {
			endpoint.ClusterID = app.Placement.ClusterID
			endpoint.Namespace = app.Placement.Namespace
			endpoint.EndpointName = app.Service.EndpointName
			endpoint.URL = endpointURL
			endpoint.Ready = ready
			endpoint.UpdatedAt = s.now().UTC()
			s.data.Endpoints[endpoint.ID] = endpoint
			app.EndpointURL = endpointURL
			return app
		}
	}
	now := s.now().UTC()
	endpoint := EndpointRegistryEntry{
		ID:                   s.nextID("endpoint"),
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Namespace:            app.Placement.Namespace,
		EndpointName:         app.Service.EndpointName,
		URL:                  endpointURL,
		Ready:                ready,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	s.data.Endpoints[endpoint.ID] = endpoint
	app.EndpointURL = endpointURL
	return app
}

func (s *FileStore) removeEndpointForServingApplicationLocked(appID string) {
	for id, endpoint := range s.data.Endpoints {
		if endpoint.ServingApplicationID == appID {
			delete(s.data.Endpoints, id)
		}
	}
}

func buildObservabilityEntry(app ServingApplication, cluster InferenceCluster) ObservabilityEntry {
	deploymentName := kubernetesName(app.Name)
	namespace := app.Placement.Namespace
	grafanaURL := cluster.GrafanaURL
	if grafanaURL != "" {
		grafanaURL = strings.TrimRight(grafanaURL, "/") + "/dashboards?var-namespace=" + namespace + "&var-deployment=" + deploymentName
	}
	return ObservabilityEntry{
		ServingApplicationID: app.ID,
		ClusterID:            app.Placement.ClusterID,
		Namespace:            namespace,
		GrafanaURL:           grafanaURL,
		PrometheusURL:        cluster.PrometheusURL,
		PrometheusQueries: []PrometheusQuery{
			{
				Name:        "frontend_request_rate",
				Description: "Approximate frontend request rate for the Serving Application.",
				Query:       `sum(rate(dynamo_frontend_requests_total{namespace="` + namespace + `"}[5m]))`,
			},
			{
				Name:        "gpu_utilization",
				Description: "GPU utilization for pods owned by the Serving Application when DCGM labels are available.",
				Query:       `avg(DCGM_FI_DEV_GPU_UTIL{namespace="` + namespace + `"})`,
			},
			{
				Name:        "pod_ready",
				Description: "Ready pod count for the Serving Application namespace.",
				Query:       `sum(kube_pod_status_ready{namespace="` + namespace + `",condition="true"})`,
			},
		},
	}
}

func defaultEndpointURL(app ServingApplication) string {
	endpointName := strings.TrimSpace(app.Service.EndpointName)
	if endpointName == "" {
		endpointName = kubernetesName(app.Name)
	}
	namespace := strings.TrimSpace(app.Placement.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	return "http://" + endpointName + "." + namespace + ".svc.cluster.local:8000/v1"
}

func taskFailureReason(task Task) string {
	if strings.TrimSpace(task.Error) != "" {
		return task.Error
	}
	return taskResultMessage(task)
}

func taskResultMessage(task Task) string {
	message, _ := task.Result["message"].(string)
	if strings.TrimSpace(message) != "" {
		return strings.TrimSpace(message)
	}
	phase, _ := task.Result["phase"].(string)
	if strings.TrimSpace(phase) != "" {
		return "task result phase: " + strings.TrimSpace(phase)
	}
	return string(task.Type) + " completed"
}

func (s *FileStore) sortedTasksLocked() []Task {
	tasks := make([]Task, 0, len(s.data.Tasks))
	for _, task := range s.data.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	return tasks
}

func (s *FileStore) nextID(prefix string) string {
	id := fmt.Sprintf("%s-%d", prefix, s.data.NextID)
	s.data.NextID++
	return id
}

func (s *FileStore) load() error {
	if s.path == "" {
		return nil
	}
	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return nil
	}
	if err := json.Unmarshal(contents, &s.data); err != nil {
		return err
	}
	normalizeStoreData(&s.data)
	return nil
}

func (s *FileStore) saveLocked() error {
	if s.persist == nil {
		return nil
	}
	return s.persist(s.data)
}

func (s *FileStore) persistFile(data storeData) error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, contents, 0o600)
}

func validateServingApplicationIntent(req CreateServingApplicationRequest, artifact ModelArtifact) error {
	if strings.TrimSpace(req.Model.Family) == "" || strings.TrimSpace(req.Model.Variant) == "" || strings.TrimSpace(req.Model.ArtifactID) == "" || strings.TrimSpace(req.Model.Quantization) == "" {
		return fmt.Errorf("%w: model family, variant, artifactId, and quantization are required", ErrInvalidInput)
	}
	if req.Model.Family != artifact.Family || req.Model.Variant != artifact.Variant || req.Model.Quantization != artifact.Quantization {
		return fmt.Errorf("%w: model intent must match model artifact family, variant, and quantization", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Placement.ClusterID) == "" || strings.TrimSpace(req.Placement.Namespace) == "" {
		return fmt.Errorf("%w: placement clusterId and namespace are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Runtime.Backend) == "" || strings.TrimSpace(req.Runtime.Topology) == "" || strings.TrimSpace(req.Runtime.Recipe) == "" {
		return fmt.Errorf("%w: runtime backend, topology, and recipe are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Service.EndpointName) == "" || strings.TrimSpace(req.Service.Protocol) == "" || strings.TrimSpace(req.Service.Exposure) == "" {
		return fmt.Errorf("%w: service endpointName, protocol, and exposure are required", ErrInvalidInput)
	}
	if strings.TrimSpace(req.Optimization.Target) == "" {
		return fmt.Errorf("%w: optimization target is required", ErrInvalidInput)
	}
	return nil
}

func isAllowedTaskType(taskType TaskType) bool {
	switch taskType {
	case TaskTypeRegisterCluster,
		TaskTypeValidateIntent,
		TaskTypePreviewDeploymentDiff,
		TaskTypeApplyDeployment,
		TaskTypeDeleteBeforeApply,
		TaskTypeInspectStatus,
		TaskTypeRetireDeployment,
		TaskTypeFetchDiagnostics,
		TaskTypeSyncEndpointReadiness:
		return true
	default:
		return false
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
