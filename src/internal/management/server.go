package management

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store      Store
	logger     *slog.Logger
	leaseTTL   time.Duration
	authConfig AuthConfig
	corsConfig CORSConfig
}

func NewServer(store Store, logger *slog.Logger) *Server {
	return NewServerWithAuth(store, logger, AuthConfig{})
}

func NewServerWithAuth(store Store, logger *slog.Logger, authConfig AuthConfig) *Server {
	return NewServerWithOptions(store, logger, authConfig, CORSConfig{})
}

func NewServerWithOptions(store Store, logger *slog.Logger, authConfig AuthConfig, corsConfig CORSConfig) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: store, logger: logger, leaseTTL: 30 * time.Second, authConfig: authConfig, corsConfig: corsConfig}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/projects", s.createProject)
	mux.HandleFunc("GET /v1/projects", s.listProjects)
	mux.HandleFunc("POST /v1/clusters", s.createCluster)
	mux.HandleFunc("GET /v1/clusters", s.listClusters)
	mux.HandleFunc("POST /v1/agents/register", s.registerAgent)
	mux.HandleFunc("GET /v1/agents", s.listAgents)
	mux.HandleFunc("POST /v1/agents/{agentID}/heartbeat", s.heartbeatAgent)
	mux.HandleFunc("POST /v1/model-artifacts", s.createModelArtifact)
	mux.HandleFunc("GET /v1/model-artifacts", s.listModelArtifacts)
	mux.HandleFunc("POST /v1/serving-applications", s.createServingApplication)
	mux.HandleFunc("GET /v1/serving-applications", s.listServingApplications)
	mux.HandleFunc("POST /v1/serving-applications/{appID}/preview-task", s.createPreviewTask)
	mux.HandleFunc("POST /v1/serving-applications/{appID}/apply-task", s.createApplyTask)
	mux.HandleFunc("POST /v1/serving-applications/{appID}/redeploy-task", s.createRedeployTask)
	mux.HandleFunc("POST /v1/serving-applications/{appID}/retire-task", s.createRetireTask)
	mux.HandleFunc("GET /v1/serving-applications/{appID}/observability", s.getObservabilityEntry)
	mux.HandleFunc("GET /v1/endpoints", s.listEndpoints)
	mux.HandleFunc("GET /v1/audit-records", s.listAuditRecords)
	mux.HandleFunc("POST /v1/tasks", s.createTask)
	mux.HandleFunc("GET /v1/tasks", s.listTasks)
	mux.HandleFunc("POST /v1/clusters/{clusterID}/tasks:lease", s.leaseTask)
	mux.HandleFunc("POST /v1/tasks/{taskID}/complete", s.completeTask)
	return requestLogger(s.logger, corsMiddleware(s.corsConfig, authMiddleware(s.authConfig, mux)))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin") {
		return
	}
	var req CreateProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.store.CreateProject(req)
	if err == nil {
		s.audit(r, "create_project", project.ID, map[string]any{"name": project.Name})
	}
	writeResult(w, project, http.StatusCreated, err)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects()
	writeResult(w, projects, http.StatusOK, err)
}

func (s *Server) createCluster(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin") {
		return
	}
	var req CreateClusterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	cluster, err := s.store.CreateCluster(req)
	if err == nil {
		s.audit(r, "create_cluster", cluster.ID, map[string]any{"name": cluster.Name})
	}
	writeResult(w, cluster, http.StatusCreated, err)
}

func (s *Server) listClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.store.ListClusters()
	writeResult(w, clusters, http.StatusOK, err)
}

func (s *Server) registerAgent(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "agent") {
		return
	}
	var req RegisterAgentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	agent, err := s.store.RegisterAgent(req)
	if err == nil {
		s.audit(r, "register_agent", agent.ID, map[string]any{"clusterId": agent.ClusterID})
	}
	writeResult(w, agent, http.StatusCreated, err)
}

func (s *Server) heartbeatAgent(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	agent, err := s.store.HeartbeatAgent(r.PathValue("agentID"), req)
	writeResult(w, agent, http.StatusOK, err)
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.store.ListAgents()
	writeResult(w, agents, http.StatusOK, err)
}

func (s *Server) createModelArtifact(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	var req CreateModelArtifactRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	artifact, err := s.store.CreateModelArtifact(req)
	if err == nil {
		s.audit(r, "create_model_artifact", artifact.ID, map[string]any{"family": artifact.Family, "variant": artifact.Variant})
	}
	writeResult(w, artifact, http.StatusCreated, err)
}

func (s *Server) listModelArtifacts(w http.ResponseWriter, r *http.Request) {
	artifacts, err := s.store.ListModelArtifacts()
	writeResult(w, artifacts, http.StatusOK, err)
}

func (s *Server) createServingApplication(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	var req CreateServingApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	app, err := s.store.CreateServingApplication(req)
	if err == nil {
		s.audit(r, "create_serving_application", app.ID, map[string]any{"projectId": app.ProjectID, "name": app.Name})
	}
	writeResult(w, app, http.StatusCreated, err)
}

func (s *Server) listServingApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListServingApplications()
	writeResult(w, apps, http.StatusOK, err)
}

func (s *Server) createPreviewTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	task, err := s.store.CreatePreviewTask(CreatePreviewTaskRequest{ServingApplicationID: r.PathValue("appID")})
	if err == nil {
		s.audit(r, "create_preview_task", task.ID, map[string]any{"servingApplicationId": r.PathValue("appID")})
	}
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createApplyTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	task, err := s.store.CreateApplyTask(CreateApplyTaskRequest{ServingApplicationID: r.PathValue("appID")})
	if err == nil {
		s.audit(r, "create_apply_task", task.ID, map[string]any{"servingApplicationId": r.PathValue("appID")})
	}
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createRedeployTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	task, err := s.store.CreateRedeployTask(CreateRedeployTaskRequest{ServingApplicationID: r.PathValue("appID")})
	if err == nil {
		s.audit(r, "create_redeploy_task", task.ID, map[string]any{"servingApplicationId": r.PathValue("appID")})
	}
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createRetireTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	task, err := s.store.CreateRetireTask(CreateRetireTaskRequest{ServingApplicationID: r.PathValue("appID")})
	if err == nil {
		s.audit(r, "create_retire_task", task.ID, map[string]any{"servingApplicationId": r.PathValue("appID")})
	}
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) getObservabilityEntry(w http.ResponseWriter, r *http.Request) {
	entry, err := s.store.GetObservabilityEntry(r.PathValue("appID"))
	writeResult(w, entry, http.StatusOK, err)
}

func (s *Server) listEndpoints(w http.ResponseWriter, r *http.Request) {
	endpoints, err := s.store.ListEndpoints()
	writeResult(w, endpoints, http.StatusOK, err)
}

func (s *Server) listAuditRecords(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin") {
		return
	}
	records, err := s.store.ListAuditRecords()
	writeResult(w, records, http.StatusOK, err)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator") {
		return
	}
	var req CreateTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.CreateTask(req)
	if err == nil {
		s.audit(r, "create_task", task.ID, map[string]any{"type": task.Type, "clusterId": task.ClusterID})
	}
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListTasks(r.URL.Query().Get("clusterId"))
	writeResult(w, tasks, http.StatusOK, err)
}

func (s *Server) leaseTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator", "agent") {
		return
	}
	var req LeaseTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.LeaseNextTask(r.PathValue("clusterID"), req, s.leaseTTL)
	writeResult(w, task, http.StatusOK, err)
}

func (s *Server) completeTask(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "admin", "operator", "agent") {
		return
	}
	var req CompleteTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.CompleteTask(r.PathValue("taskID"), req)
	if err == nil {
		s.audit(r, "complete_task", task.ID, map[string]any{"status": task.Status, "type": task.Type})
	}
	writeResult(w, task, http.StatusOK, err)
}

func (s *Server) audit(r *http.Request, action string, resource string, metadata map[string]any) {
	actor := ActorFromContext(r.Context())
	if _, err := s.store.RecordAudit(actor.Name, action, resource, metadata); err != nil {
		s.logger.Error("record audit", "error", err, "action", action, "resource", resource)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func writeResult(w http.ResponseWriter, value any, successStatus int, err error) {
	if err == nil {
		writeJSON(w, successStatus, value)
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrTaskLeaseHeld):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/healthz") {
			logger.Info("request", "method", r.Method, "path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
