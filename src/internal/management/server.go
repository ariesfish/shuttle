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
	store    Store
	logger   *slog.Logger
	leaseTTL time.Duration
}

func NewServer(store Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: store, logger: logger, leaseTTL: 30 * time.Second}
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
	mux.HandleFunc("POST /v1/tasks", s.createTask)
	mux.HandleFunc("GET /v1/tasks", s.listTasks)
	mux.HandleFunc("POST /v1/clusters/{clusterID}/tasks:lease", s.leaseTask)
	mux.HandleFunc("POST /v1/tasks/{taskID}/complete", s.completeTask)
	return requestLogger(s.logger, mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := s.store.CreateProject(req)
	writeResult(w, project, http.StatusCreated, err)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects()
	writeResult(w, projects, http.StatusOK, err)
}

func (s *Server) createCluster(w http.ResponseWriter, r *http.Request) {
	var req CreateClusterRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	cluster, err := s.store.CreateCluster(req)
	writeResult(w, cluster, http.StatusCreated, err)
}

func (s *Server) listClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.store.ListClusters()
	writeResult(w, clusters, http.StatusOK, err)
}

func (s *Server) registerAgent(w http.ResponseWriter, r *http.Request) {
	var req RegisterAgentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	agent, err := s.store.RegisterAgent(req)
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
	var req CreateModelArtifactRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	artifact, err := s.store.CreateModelArtifact(req)
	writeResult(w, artifact, http.StatusCreated, err)
}

func (s *Server) listModelArtifacts(w http.ResponseWriter, r *http.Request) {
	artifacts, err := s.store.ListModelArtifacts()
	writeResult(w, artifacts, http.StatusOK, err)
}

func (s *Server) createServingApplication(w http.ResponseWriter, r *http.Request) {
	var req CreateServingApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	app, err := s.store.CreateServingApplication(req)
	writeResult(w, app, http.StatusCreated, err)
}

func (s *Server) listServingApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListServingApplications()
	writeResult(w, apps, http.StatusOK, err)
}

func (s *Server) createPreviewTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.CreatePreviewTask(CreatePreviewTaskRequest{ServingApplicationID: r.PathValue("appID")})
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createApplyTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.CreateApplyTask(CreateApplyTaskRequest{ServingApplicationID: r.PathValue("appID")})
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createRedeployTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.CreateRedeployTask(CreateRedeployTaskRequest{ServingApplicationID: r.PathValue("appID")})
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) createRetireTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.CreateRetireTask(CreateRetireTaskRequest{ServingApplicationID: r.PathValue("appID")})
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

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.CreateTask(req)
	writeResult(w, task, http.StatusCreated, err)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListTasks(r.URL.Query().Get("clusterId"))
	writeResult(w, tasks, http.StatusOK, err)
}

func (s *Server) leaseTask(w http.ResponseWriter, r *http.Request) {
	var req LeaseTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.LeaseNextTask(r.PathValue("clusterID"), req, s.leaseTTL)
	writeResult(w, task, http.StatusOK, err)
}

func (s *Server) completeTask(w http.ResponseWriter, r *http.Request) {
	var req CompleteTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	task, err := s.store.CompleteTask(r.PathValue("taskID"), req)
	writeResult(w, task, http.StatusOK, err)
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
