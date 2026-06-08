export interface InferenceCluster {
  id: string;
  name: string;
  description?: string;
  prometheusUrl?: string;
  grafanaUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ClusterAgent {
  id: string;
  clusterId: string;
  version?: string;
  capabilities?: Record<string, string>;
  lastHeartbeat?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateClusterInput {
  name: string;
  description?: string;
  prometheusUrl?: string;
  grafanaUrl?: string;
}

export interface Project {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateProjectInput {
  name: string;
}

export interface ModelArtifact {
  id: string;
  family: string;
  variant: string;
  revision: string;
  pvcName?: string;
  pvcMountPath: string;
  pvcModelPath: string;
  hostCachePath?: string;
  quantization: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateModelArtifactInput {
  family: string;
  variant: string;
  revision: string;
  pvcName?: string;
  pvcMountPath: string;
  pvcModelPath: string;
  hostCachePath?: string;
  quantization: string;
}

export interface ServingApplication {
  id: string;
  projectId: string;
  name: string;
  model: ModelIntent;
  placement: PlacementIntent;
  runtime: RuntimeIntent;
  service: ServiceIntent;
  optimization: OptimizationIntent;
  desiredState: string;
  phase: string;
  activeVersion: number;
  endpointUrl?: string;
  grafanaUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ServingApplicationTransition {
  id: string;
  servingApplicationId: string;
  actor: string;
  taskId?: string;
  from?: string;
  to: string;
  reason?: string;
  createdAt: string;
}

export interface ModelIntent {
  family: string;
  variant: string;
  artifactId: string;
  quantization: string;
}

export interface PlacementIntent {
  clusterId: string;
  acceleratorPoolId?: string;
  namespace: string;
}

export interface RuntimeIntent {
  backend: string;
  topology: string;
  recipe: string;
  replicas?: Record<string, number>;
}

export interface ServiceIntent {
  endpointName: string;
  protocol: string;
  exposure: string;
}

export interface OptimizationIntent {
  target: string;
  ttftMs?: number;
  itlMs?: number;
  profilingMode: string;
}

export interface CreateServingApplicationInput {
  projectId: string;
  name: string;
  model: ModelIntent;
  placement: PlacementIntent;
  runtime: RuntimeIntent;
  service: ServiceIntent;
  optimization: OptimizationIntent;
}

export interface Task {
  id: string;
  clusterId: string;
  type: string;
  status: string;
  payload?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export interface EndpointRegistryEntry {
  id: string;
  servingApplicationId: string;
  clusterId: string;
  namespace: string;
  endpointName: string;
  url: string;
  ready: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ObservabilityEntry {
  servingApplicationId: string;
  clusterId: string;
  namespace: string;
  grafanaUrl?: string;
  prometheusUrl?: string;
  prometheusQueries: Array<{ name: string; description: string; query: string }>;
}

export interface AuditRecord {
  id: string;
  actor: string;
  action: string;
  resource: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
}

const defaultBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export function getApiSettings() {
  return {
    baseUrl: localStorage.getItem('apiBaseUrl') || defaultBaseUrl,
    token: localStorage.getItem('authToken') || '',
  };
}

export function saveApiSettings(baseUrl: string, token: string) {
  localStorage.setItem('apiBaseUrl', baseUrl.trim() || defaultBaseUrl);
  localStorage.setItem('authToken', token.trim());
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const { baseUrl, token } = getApiSettings();
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  headers.set('X-Actor', 'web-console');
  headers.set('X-Role', 'admin');
  const response = await fetch(`${baseUrl}${path}`, { ...init, headers });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error || response.statusText);
  }
  return response.json() as Promise<T>;
}

export const api = {
  listClusters: () => request<InferenceCluster[]>('/v1/clusters'),
  createCluster: (input: CreateClusterInput) => request<InferenceCluster>('/v1/clusters', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  listAgents: () => request<ClusterAgent[]>('/v1/agents'),
  listProjects: () => request<Project[]>('/v1/projects'),
  createProject: (input: CreateProjectInput) => request<Project>('/v1/projects', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  listModelArtifacts: () => request<ModelArtifact[]>('/v1/model-artifacts'),
  createModelArtifact: (input: CreateModelArtifactInput) => request<ModelArtifact>('/v1/model-artifacts', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  listServingApplications: () => request<ServingApplication[]>('/v1/serving-applications'),
  createServingApplication: (input: CreateServingApplicationInput) => request<ServingApplication>('/v1/serving-applications', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  createPreviewTask: (appId: string) => request<Task>(`/v1/serving-applications/${appId}/preview-task`, { method: 'POST' }),
  createApplyTask: (appId: string) => request<Task>(`/v1/serving-applications/${appId}/apply-task`, { method: 'POST' }),
  createRedeployTask: (appId: string) => request<Task>(`/v1/serving-applications/${appId}/redeploy-task`, { method: 'POST' }),
  createRetireTask: (appId: string) => request<Task>(`/v1/serving-applications/${appId}/retire-task`, { method: 'POST' }),
  createDiagnosticsTask: (appId: string) => request<Task>(`/v1/serving-applications/${appId}/diagnostics-task`, { method: 'POST' }),
  listServingApplicationTransitions: (appId: string) => request<ServingApplicationTransition[]>(`/v1/serving-applications/${appId}/transitions`),
  listTasks: () => request<Task[]>('/v1/tasks'),
  listEndpoints: () => request<EndpointRegistryEntry[]>('/v1/endpoints'),
  getObservabilityEntry: (appId: string) => request<ObservabilityEntry>(`/v1/serving-applications/${appId}/observability`),
  listAuditRecords: () => request<AuditRecord[]>('/v1/audit-records'),
};
