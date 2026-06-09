package management

import (
	"fmt"
	"strings"

	platformtask "zhiliu/internal/task"
)

func (s *FileStore) LoadActionState(appID string) (ServingApplicationActionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.data.ServingApplications[appID]
	if !ok {
		return ServingApplicationActionState{}, ErrNotFound
	}
	artifact, ok := s.data.ModelArtifacts[app.Model.ArtifactID]
	if !ok {
		return ServingApplicationActionState{}, fmt.Errorf("%w: model artifact does not exist", ErrInvalidInput)
	}
	recipe, ok := s.recipes.Get(app.Runtime.Recipe)
	if !ok {
		return ServingApplicationActionState{}, fmt.Errorf("%w: unsupported recipe", ErrInvalidInput)
	}
	cluster, ok := s.data.Clusters[app.Placement.ClusterID]
	if !ok {
		return ServingApplicationActionState{}, fmt.Errorf("%w: cluster does not exist", ErrInvalidInput)
	}
	return ServingApplicationActionState{App: app, Artifact: artifact, Recipe: recipe, Cluster: cluster}, nil
}

func (s *FileStore) SaveRequestedAction(request ServingApplicationRequestedAction) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := s.newTaskFromEnvelopeLocked(request.Task)
	if request.TransitionPhase != "" {
		actor := request.Actor
		if actor == "" {
			actor = "system"
		}
		s.setServingApplicationPhaseLocked(request.Task.Payload.ServingApplicationID(), actor, task.ID, request.TransitionPhase, request.TransitionReason)
	}
	s.data.Tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *FileStore) LoadCompletionState(taskID string) (ServingApplicationCompletionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.data.Tasks[taskID]
	if !ok {
		return ServingApplicationCompletionState{}, ErrNotFound
	}
	appID, err := servingApplicationIDForTask(platformtask.DefaultRegistry(), task)
	if err != nil {
		return ServingApplicationCompletionState{Task: task}, nil
	}
	app, ok := s.data.ServingApplications[appID]
	if !ok {
		return ServingApplicationCompletionState{Task: task}, nil
	}
	return ServingApplicationCompletionState{App: app, Task: task}, nil
}

func (s *FileStore) SaveAcceptedCompletion(accepted ServingApplicationAcceptedCompletion) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := accepted.Task
	s.data.Tasks[task.ID] = task
	if accepted.Phase != "" && accepted.Task.Payload != nil {
		appID, err := servingApplicationIDForTask(platformtask.DefaultRegistry(), task)
		if err == nil {
			s.setServingApplicationPhaseLocked(appID, task.LeaseOwner, task.ID, accepted.Phase, accepted.Reason)
			s.applyEndpointOperationLocked(appID, accepted)
		}
	}
	return task, s.saveLocked()
}

func (s *FileStore) applyEndpointOperationLocked(appID string, accepted ServingApplicationAcceptedCompletion) {
	switch accepted.EndpointOperation {
	case EndpointOperationUpsert:
		updatedApp := s.data.ServingApplications[appID]
		updatedApp = s.upsertEndpointLocked(updatedApp, accepted.Endpoint)
		updatedApp.UpdatedAt = s.now().UTC()
		s.data.ServingApplications[updatedApp.ID] = updatedApp
	case EndpointOperationRemove:
		s.removeEndpointForServingApplicationLocked(appID)
	}
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

func (s *FileStore) newTaskFromEnvelopeLocked(envelope platformtask.Envelope) Task {
	return s.newTaskLocked(CreateTaskRequest{
		ClusterID: envelope.ClusterID,
		Type:      envelope.Type,
		Payload:   platformtask.EncodePayload(envelope.Payload),
	})
}

func (s *FileStore) upsertEndpointLocked(app ServingApplication, planned EndpointRegistryEntry) ServingApplication {
	planned.ServingApplicationID = app.ID
	for _, endpoint := range s.data.Endpoints {
		if endpoint.ServingApplicationID == app.ID {
			endpoint.ClusterID = planned.ClusterID
			endpoint.Namespace = planned.Namespace
			endpoint.EndpointName = planned.EndpointName
			endpoint.URL = planned.URL
			endpoint.Ready = planned.Ready
			endpoint.UpdatedAt = s.now().UTC()
			s.data.Endpoints[endpoint.ID] = endpoint
			app.EndpointURL = planned.URL
			return app
		}
	}
	now := s.now().UTC()
	planned.ID = s.nextID("endpoint")
	planned.CreatedAt = now
	planned.UpdatedAt = now
	s.data.Endpoints[planned.ID] = planned
	app.EndpointURL = planned.URL
	return app
}

func (s *FileStore) removeEndpointForServingApplicationLocked(appID string) {
	for id, endpoint := range s.data.Endpoints {
		if endpoint.ServingApplicationID == appID {
			delete(s.data.Endpoints, id)
		}
	}
}
