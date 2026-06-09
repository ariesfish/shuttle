package management

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	platformtask "zhiliu/internal/task"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrTaskLeaseHeld = errors.New("task lease held by another agent")
)

// ManagementStore is the broad Phase 1 persistence Adapter used by the HTTP
// Management Plane. Domain modules should prefer narrower seams such as
// ServingApplicationRepository when they only need lifecycle behavior.
type ManagementStore interface {
	ProjectStore
	ClusterStore
	ClusterAgentStore
	AcceleratorInventoryStore
	AcceleratorPoolStore
	ModelArtifactStore
	RecipeStore
	ServingApplicationStore
	ObservabilityStore
	AuditStore
	TaskStore
	ServingApplicationRepository
}

type ProjectStore interface {
	CreateProject(CreateProjectRequest) (Project, error)
	ListProjects() ([]Project, error)
}

type ClusterStore interface {
	CreateCluster(CreateClusterRequest) (InferenceCluster, error)
	ListClusters() ([]InferenceCluster, error)
	GetCluster(string) (InferenceCluster, error)
}

type ClusterAgentStore interface {
	RegisterAgent(RegisterAgentRequest) (ClusterAgent, error)
	HeartbeatAgent(string, HeartbeatRequest) (ClusterAgent, error)
	ListAgents() ([]ClusterAgent, error)
}

type AcceleratorInventoryStore interface {
	ReportAcceleratorInventory(clusterID string, req ReportAcceleratorInventoryRequest) (AcceleratorInventory, error)
	GetAcceleratorInventory(clusterID string) (AcceleratorInventory, error)
	ListAcceleratorInventoryRevisions(clusterID string) ([]AcceleratorInventory, error)
}

type AcceleratorPoolStore interface {
	CreateAcceleratorPool(CreateAcceleratorPoolRequest) (AcceleratorPool, error)
	ListAcceleratorPools(clusterID string) ([]AcceleratorPool, error)
	ListAcceleratorPoolSummaries(clusterID string) ([]AcceleratorPoolSummary, error)
}

type ModelArtifactStore interface {
	CreateModelArtifact(CreateModelArtifactRequest) (ModelArtifact, error)
	ListModelArtifacts() ([]ModelArtifact, error)
	GetModelArtifact(string) (ModelArtifact, error)
}

type RecipeStore interface {
	ListRecipes() ([]ServingRecipe, error)
	ListServingApplicationCreationPlans(artifactID string) ([]ServingApplicationCreationPlan, error)
}

type ServingApplicationStore interface {
	CreateServingApplication(CreateServingApplicationRequest) (ServingApplication, error)
	ListServingApplications() ([]ServingApplication, error)
	GetServingApplication(string) (ServingApplication, error)
	ListServingApplicationTransitions(string) ([]ServingApplicationTransition, error)
	ListEndpoints() ([]EndpointRegistryEntry, error)
}

type ObservabilityStore interface {
	GetObservabilityEntry(string) (ObservabilityEntry, error)
}

type AuditStore interface {
	ListAuditRecords() ([]AuditRecord, error)
	RecordAudit(actor, action, resource string, metadata map[string]any) (AuditRecord, error)
}

type TaskStore interface {
	CreateTask(CreateTaskRequest) (Task, error)
	ListTasks(clusterID string) ([]Task, error)
	LeaseNextTask(clusterID string, req LeaseTaskRequest, ttl time.Duration) (Task, error)
	RenewTaskLease(taskID string, req RenewTaskLeaseRequest, ttl time.Duration) (Task, error)
}

type FileStore struct {
	mu      sync.Mutex
	path    string
	data    storeData
	now     func() time.Time
	persist func(storeData) error
	recipes *RecipeRegistry
}

type storeData struct {
	NextID                        int                                     `json:"nextId"`
	Projects                      map[string]Project                      `json:"projects"`
	Clusters                      map[string]InferenceCluster             `json:"clusters"`
	Agents                        map[string]ClusterAgent                 `json:"agents"`
	AcceleratorPools              map[string]AcceleratorPool              `json:"acceleratorPools"`
	ModelArtifacts                map[string]ModelArtifact                `json:"modelArtifacts"`
	ServingApplications           map[string]ServingApplication           `json:"servingApplications"`
	Transitions                   map[string]ServingApplicationTransition `json:"transitions"`
	Endpoints                     map[string]EndpointRegistryEntry        `json:"endpoints"`
	AuditRecords                  map[string]AuditRecord                  `json:"auditRecords"`
	Tasks                         map[string]Task                         `json:"tasks"`
	AcceleratorInventory          map[string]AcceleratorInventory         `json:"acceleratorInventory"`
	AcceleratorInventoryRevisions map[string][]AcceleratorInventory       `json:"acceleratorInventoryRevisions"`
}

func NewFileStore(path string) (*FileStore, error) {
	return NewFileStoreWithRecipes(path, MustLoadDefaultRecipeRegistry())
}

func NewFileStoreWithRecipes(path string, recipes *RecipeRegistry) (*FileStore, error) {
	store := &FileStore{path: path, now: time.Now, recipes: recipes}
	store.persist = store.persistFile
	store.data = newStoreData()
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func newStoreData() storeData {
	return storeData{
		NextID:                        1,
		Projects:                      map[string]Project{},
		Clusters:                      map[string]InferenceCluster{},
		Agents:                        map[string]ClusterAgent{},
		AcceleratorPools:              map[string]AcceleratorPool{},
		ModelArtifacts:                map[string]ModelArtifact{},
		ServingApplications:           map[string]ServingApplication{},
		Transitions:                   map[string]ServingApplicationTransition{},
		Endpoints:                     map[string]EndpointRegistryEntry{},
		AuditRecords:                  map[string]AuditRecord{},
		Tasks:                         map[string]Task{},
		AcceleratorInventory:          map[string]AcceleratorInventory{},
		AcceleratorInventoryRevisions: map[string][]AcceleratorInventory{},
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
	if strings.TrimSpace(req.LastInventoryRevision) != "" {
		agent.LastInventoryRevision = strings.TrimSpace(req.LastInventoryRevision)
	}
	if strings.TrimSpace(req.LastInventoryFreshness) != "" {
		agent.LastInventoryFreshness = strings.TrimSpace(req.LastInventoryFreshness)
	}
	if !req.LastInventoryObservedAt.IsZero() {
		agent.LastInventoryObservedAt = req.LastInventoryObservedAt.UTC()
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

func (s *FileStore) ReportAcceleratorInventory(clusterID string, req ReportAcceleratorInventoryRequest) (AcceleratorInventory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.data.Agents[req.AgentID]
	if !ok {
		return AcceleratorInventory{}, ErrNotFound
	}
	if agent.ClusterID != clusterID {
		return AcceleratorInventory{}, fmt.Errorf("%w: agent does not belong to cluster", ErrInvalidInput)
	}
	if _, ok := s.data.Clusters[clusterID]; !ok {
		return AcceleratorInventory{}, ErrNotFound
	}
	schemaVersion := strings.TrimSpace(req.SchemaVersion)
	if schemaVersion == "" {
		return AcceleratorInventory{}, fmt.Errorf("%w: schemaVersion is required", ErrInvalidInput)
	}
	observedAt := req.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = s.now().UTC()
	}
	nodes := cloneInventoryNodes(req.Nodes)
	for index := range nodes {
		if nodes[index].ObservedAt.IsZero() {
			nodes[index].ObservedAt = observedAt
		} else {
			nodes[index].ObservedAt = nodes[index].ObservedAt.UTC()
		}
	}
	now := s.now().UTC()
	inventory := AcceleratorInventory{
		ClusterID:          clusterID,
		AgentID:            req.AgentID,
		SchemaVersion:      schemaVersion,
		Revision:           strings.TrimSpace(req.Revision),
		ObservedAt:         observedAt,
		ReportedAt:         now,
		Freshness:          AcceleratorInventoryFreshnessFresh,
		Nodes:              nodes,
		ProbeStatuses:      cloneInventoryProbes(req.ProbeStatuses),
		CollectionMetadata: cloneStringMap(req.CollectionMetadata),
	}
	if inventory.Revision == "" {
		inventory.Revision = acceleratorInventoryRevision(inventory)
	}
	previous := s.data.AcceleratorInventory[clusterID]
	revisions := s.data.AcceleratorInventoryRevisions[clusterID]
	if previous.Revision != inventory.Revision {
		revisions = append([]AcceleratorInventory{inventory}, revisions...)
		if len(revisions) > 10 {
			revisions = revisions[:10]
		}
		s.data.AcceleratorInventoryRevisions[clusterID] = revisions
		s.recordAuditLocked(req.AgentID, "accelerator_inventory.report", "cluster:"+clusterID, map[string]any{"revision": inventory.Revision, "previousRevision": previous.Revision})
	}
	inventory.RevisionCount = len(revisions)
	s.data.AcceleratorInventory[clusterID] = inventory
	agent.LastInventoryRevision = inventory.Revision
	agent.LastInventoryFreshness = string(inventory.Freshness)
	agent.LastInventoryObservedAt = inventory.ObservedAt
	agent.LastInventoryReportedAt = inventory.ReportedAt
	agent.UpdatedAt = now
	s.data.Agents[agent.ID] = agent
	return inventory, s.saveLocked()
}

func (s *FileStore) GetAcceleratorInventory(clusterID string) (AcceleratorInventory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Clusters[clusterID]; !ok {
		return AcceleratorInventory{}, ErrNotFound
	}
	inventory, ok := s.data.AcceleratorInventory[clusterID]
	if !ok {
		freshness := AcceleratorInventoryFreshnessUnsupported
		for _, agent := range s.data.Agents {
			if agent.ClusterID == clusterID && strings.TrimSpace(agent.LastInventoryFreshness) != "" {
				freshness = AcceleratorInventoryFreshnessMissing
				break
			}
		}
		return AcceleratorInventory{ClusterID: clusterID, Freshness: freshness}, nil
	}
	inventory.RevisionCount = len(s.data.AcceleratorInventoryRevisions[clusterID])
	if isInventoryStale(s.now().UTC(), inventory.ReportedAt) {
		inventory.Freshness = AcceleratorInventoryFreshnessStale
	}
	return inventory, nil
}

func (s *FileStore) ListAcceleratorInventoryRevisions(clusterID string) ([]AcceleratorInventory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Clusters[clusterID]; !ok {
		return nil, ErrNotFound
	}
	revisions := s.data.AcceleratorInventoryRevisions[clusterID]
	output := make([]AcceleratorInventory, len(revisions))
	copy(output, revisions)
	return output, nil
}

func (s *FileStore) CreateAcceleratorPool(req CreateAcceleratorPoolRequest) (AcceleratorPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	clusterID := strings.TrimSpace(req.ClusterID)
	if name == "" || clusterID == "" {
		return AcceleratorPool{}, fmt.Errorf("%w: name and clusterId are required", ErrInvalidInput)
	}
	if _, ok := s.data.Clusters[clusterID]; !ok {
		return AcceleratorPool{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}
	now := s.now().UTC()
	pool := AcceleratorPool{ID: s.nextID("pool"), ClusterID: clusterID, Name: name, Description: strings.TrimSpace(req.Description), NodeSelector: cloneStringMap(req.NodeSelector), CreatedAt: now, UpdatedAt: now}
	s.data.AcceleratorPools[pool.ID] = pool
	return pool, s.saveLocked()
}

func (s *FileStore) ListAcceleratorPools(clusterID string) ([]AcceleratorPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pools := make([]AcceleratorPool, 0, len(s.data.AcceleratorPools))
	for _, pool := range s.data.AcceleratorPools {
		if strings.TrimSpace(clusterID) == "" || pool.ClusterID == clusterID {
			pool.NodeSelector = cloneStringMap(pool.NodeSelector)
			pools = append(pools, pool)
		}
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i].CreatedAt.Before(pools[j].CreatedAt) })
	return pools, nil
}

func (s *FileStore) ListAcceleratorPoolSummaries(clusterID string) ([]AcceleratorPoolSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pools := make([]AcceleratorPool, 0, len(s.data.AcceleratorPools))
	for _, pool := range s.data.AcceleratorPools {
		if strings.TrimSpace(clusterID) == "" || pool.ClusterID == clusterID {
			pools = append(pools, pool)
		}
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i].CreatedAt.Before(pools[j].CreatedAt) })
	summaries := make([]AcceleratorPoolSummary, 0, len(pools))
	for _, pool := range pools {
		inventory := s.data.AcceleratorInventory[pool.ClusterID]
		if inventory.ClusterID == "" {
			summaries = append(summaries, AcceleratorPoolSummary{Pool: pool, Freshness: AcceleratorInventoryFreshnessMissing, Warnings: []string{"inventory missing"}})
			continue
		}
		if isInventoryStale(s.now().UTC(), inventory.ReportedAt) {
			inventory.Freshness = AcceleratorInventoryFreshnessStale
		}
		summaries = append(summaries, summarizeAcceleratorPool(pool, inventory))
	}
	return summaries, nil
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

func (s *FileStore) ListRecipes() ([]ServingRecipe, error) {
	return s.recipes.List(), nil
}

func (s *FileStore) ListServingApplicationCreationPlans(artifactID string) ([]ServingApplicationCreationPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	artifact, ok := s.data.ModelArtifacts[artifactID]
	if !ok {
		return nil, ErrNotFound
	}
	return s.recipes.CreationPlans(artifact), nil
}

func (s *FileStore) validateInventoryCompatibilityLocked(req CreateServingApplicationRequest) (string, error) {
	inventory, ok := s.data.AcceleratorInventory[req.Placement.ClusterID]
	if !ok {
		if strings.TrimSpace(req.Placement.AcceleratorPoolID) == "" && !s.clusterReportsInventoryLocked(req.Placement.ClusterID) {
			return "", nil
		}
		return "", fmt.Errorf("%w: accelerator inventory missing for cluster %s", ErrInvalidInput, req.Placement.ClusterID)
	}
	if isInventoryStale(s.now().UTC(), inventory.ReportedAt) {
		return inventory.Revision, fmt.Errorf("%w: accelerator inventory stale for cluster %s revision %s", ErrInvalidInput, req.Placement.ClusterID, inventory.Revision)
	}
	if inventory.Freshness != AcceleratorInventoryFreshnessFresh {
		return inventory.Revision, fmt.Errorf("%w: accelerator inventory is %s for cluster %s", ErrInvalidInput, inventory.Freshness, req.Placement.ClusterID)
	}
	nodes := inventory.Nodes
	if strings.TrimSpace(req.Placement.AcceleratorPoolID) != "" {
		pool, ok := s.data.AcceleratorPools[req.Placement.AcceleratorPoolID]
		if !ok || pool.ClusterID != req.Placement.ClusterID {
			return inventory.Revision, fmt.Errorf("%w: accelerator pool %s does not exist for cluster %s", ErrInvalidInput, req.Placement.AcceleratorPoolID, req.Placement.ClusterID)
		}
		filtered := make([]AcceleratorInventoryNode, 0, len(nodes))
		for _, node := range nodes {
			if nodeMatchesSelector(node, pool.NodeSelector) {
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}
	if len(nodes) == 0 {
		return inventory.Revision, fmt.Errorf("%w: accelerator inventory revision %s has no nodes matching placement", ErrInvalidInput, inventory.Revision)
	}
	if requiresLargeNVIDIA(req) {
		if err := validateLargeNVIDIAInventory(req, inventory.Revision, nodes); err != nil {
			return inventory.Revision, err
		}
	}
	return inventory.Revision, nil
}

func (s *FileStore) clusterReportsInventoryLocked(clusterID string) bool {
	for _, agent := range s.data.Agents {
		if agent.ClusterID == clusterID && strings.TrimSpace(agent.LastInventoryFreshness) != "" {
			return true
		}
	}
	return false
}

func requiresLargeNVIDIA(req CreateServingApplicationRequest) bool {
	return strings.EqualFold(req.Model.Family, "deepseek-v4") && strings.EqualFold(req.Model.Variant, "flash")
}

func validateLargeNVIDIAInventory(req CreateServingApplicationRequest, revision string, nodes []AcceleratorInventoryNode) error {
	for _, node := range nodes {
		gpuCount := 0
		memoryMiB := 0
		model := ""
		for _, accelerator := range node.Accelerators {
			if accelerator.Vendor != "nvidia" {
				continue
			}
			gpuCount += accelerator.DeviceCount
			if accelerator.MemoryMiB > memoryMiB {
				memoryMiB = accelerator.MemoryMiB
			}
			if model == "" {
				model = accelerator.Product
			}
		}
		if gpuCount < 8 {
			continue
		}
		if memoryMiB < 81920 {
			return fmt.Errorf("%w: accelerator inventory revision %s node %s has insufficient NVIDIA memoryMiB=%d, need >=81920", ErrInvalidInput, revision, node.Name, memoryMiB)
		}
		if strings.EqualFold(req.Runtime.Topology, "pd-disagg") && !nodeHasConnectivity(node, "rdma") {
			return fmt.Errorf("%w: accelerator inventory revision %s node %s missing RDMA connectivity for topology %s", ErrInvalidInput, revision, node.Name, req.Runtime.Topology)
		}
		if model == "" {
			model = "unknown NVIDIA"
		}
		return nil
	}
	return fmt.Errorf("%w: accelerator inventory revision %s has insufficient NVIDIA GPU count, need >=8 per node", ErrInvalidInput, revision)
}

func nodeHasConnectivity(node AcceleratorInventoryNode, kind string) bool {
	for _, fact := range node.Connectivity {
		if fact.Type == kind && fact.Present {
			return true
		}
	}
	return false
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
	if _, err := s.recipes.ValidateIntent(req, artifact); err != nil {
		return ServingApplication{}, err
	}
	inventoryRevision, err := s.validateInventoryCompatibilityLocked(req)
	if err != nil {
		return ServingApplication{}, err
	}

	now := s.now().UTC()
	app := ServingApplication{
		ID:                          s.nextID("app"),
		ProjectID:                   req.ProjectID,
		Name:                        name,
		Model:                       req.Model,
		Placement:                   req.Placement,
		Runtime:                     req.Runtime,
		Service:                     req.Service,
		Optimization:                req.Optimization,
		DesiredState:                "Active",
		Phase:                       ServingApplicationPhaseDraft,
		ActiveVersion:               1,
		ValidationInventoryRevision: inventoryRevision,
		CreatedAt:                   now,
		UpdatedAt:                   now,
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

	record := s.recordAuditLocked(actor, action, resource, metadata)
	return record, s.saveLocked()
}

func (s *FileStore) recordAuditLocked(actor, action, resource string, metadata map[string]any) AuditRecord {
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
	return record
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

func summarizeAcceleratorPool(pool AcceleratorPool, inventory AcceleratorInventory) AcceleratorPoolSummary {
	summary := AcceleratorPoolSummary{Pool: pool, Freshness: inventory.Freshness, InventoryRevision: inventory.Revision, AcceleratorModels: map[string]int{}, MemoryMiBSummary: map[string]int{}, Labels: map[string][]string{}}
	for _, node := range inventory.Nodes {
		if !nodeMatchesSelector(node, pool.NodeSelector) {
			continue
		}
		summary.NodeCount++
		for key, value := range node.Labels {
			entry := key + "=" + value
			summary.Labels[key] = append(summary.Labels[key], entry)
		}
		summary.Taints = append(summary.Taints, node.Taints...)
		for _, accelerator := range node.Accelerators {
			summary.AcceleratorCount += accelerator.DeviceCount
			model := strings.TrimSpace(accelerator.Product)
			if model == "" {
				model = accelerator.Vendor + ":unknown"
			}
			summary.AcceleratorModels[model] += accelerator.DeviceCount
			if accelerator.MemoryMiB > 0 {
				summary.MemoryMiBSummary[model] = accelerator.MemoryMiB
			}
		}
	}
	if summary.NodeCount == 0 {
		summary.Warnings = append(summary.Warnings, "pool selector matched no observed nodes")
	}
	if summary.Freshness == AcceleratorInventoryFreshnessStale {
		summary.Warnings = append(summary.Warnings, "inventory stale")
	}
	if len(summary.AcceleratorModels) == 0 {
		summary.AcceleratorModels = nil
	}
	if len(summary.MemoryMiBSummary) == 0 {
		summary.MemoryMiBSummary = nil
	}
	if len(summary.Labels) == 0 {
		summary.Labels = nil
	}
	if len(summary.Taints) == 0 {
		summary.Taints = nil
	} else {
		sort.Strings(summary.Taints)
	}
	return summary
}

func nodeMatchesSelector(node AcceleratorInventoryNode, selector map[string]string) bool {
	for key, expected := range selector {
		actual, ok := node.Labels[key]
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func isInventoryStale(now time.Time, reportedAt time.Time) bool {
	return !reportedAt.IsZero() && now.Sub(reportedAt.UTC()) > 5*time.Minute
}

func acceleratorInventoryRevision(inventory AcceleratorInventory) string {
	copy := inventory
	copy.Revision = ""
	copy.ReportedAt = time.Time{}
	contents, err := json.Marshal(copy)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

func cloneInventoryNodes(input []AcceleratorInventoryNode) []AcceleratorInventoryNode {
	if input == nil {
		return nil
	}
	output := make([]AcceleratorInventoryNode, 0, len(input))
	for _, node := range input {
		accelerators := make([]AcceleratorInventoryAccelerator, 0, len(node.Accelerators))
		for _, accelerator := range node.Accelerators {
			accelerator.VendorDetails = cloneStringMap(accelerator.VendorDetails)
			accelerators = append(accelerators, accelerator)
		}
		node.Labels = cloneStringMap(node.Labels)
		node.Taints = append([]string(nil), node.Taints...)
		node.Capacity = cloneStringMap(node.Capacity)
		node.Allocatable = cloneStringMap(node.Allocatable)
		node.AcceleratorResourceNames = append([]string(nil), node.AcceleratorResourceNames...)
		node.Accelerators = accelerators
		connectivity := make([]AcceleratorInventoryConnectivity, 0, len(node.Connectivity))
		for _, item := range node.Connectivity {
			item.Details = cloneStringMap(item.Details)
			connectivity = append(connectivity, item)
		}
		node.Connectivity = connectivity
		output = append(output, node)
	}
	return output
}

func cloneInventoryProbes(input []AcceleratorInventoryProbe) []AcceleratorInventoryProbe {
	if input == nil {
		return nil
	}
	output := make([]AcceleratorInventoryProbe, len(input))
	copy(output, input)
	return output
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

func isAllowedTaskType(taskType platformtask.TaskType) bool {
	return platformtask.IsWhitelisted(taskType)
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
