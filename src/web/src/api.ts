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
  lastInventoryRevision?: string;
  lastInventoryFreshness?: string;
  lastInventoryObservedAt?: string;
  lastInventoryReportedAt?: string;
  lastHeartbeat?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AcceleratorInventory {
  clusterId: string;
  agentId?: string;
  schemaVersion?: string;
  revision?: string;
  observedAt?: string;
  reportedAt?: string;
  freshness: 'fresh' | 'missing' | 'unsupported' | string;
  nodes?: AcceleratorInventoryNode[];
  probeStatuses?: AcceleratorInventoryProbe[];
  collectionMetadata?: Record<string, string>;
}

export interface AcceleratorInventoryNode {
  name: string;
  labels?: Record<string, string>;
  taints?: string[];
  capacity?: Record<string, string>;
  allocatable?: Record<string, string>;
  acceleratorResourceNames?: string[];
  accelerators?: AcceleratorInventoryAccelerator[];
  observedAt?: string;
}

export interface AcceleratorInventoryAccelerator {
  vendor: string;
  product?: string;
  deviceCount?: number;
  memoryMiB?: number;
  vendorDetails?: Record<string, string>;
}

export interface AcceleratorInventoryProbe {
  name: string;
  status: string;
  message?: string;
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

export interface ServingRecipe {
  apiVersion: string;
  kind: string;
  metadata: { id: string; name: string; description?: string };
  spec: {
    model: { family: string; variants: string[]; quantizations: string[] };
    runtime: { backend: string; topology: string };
    support: { status: 'supported' | 'experimental' | 'blocked'; warning?: string; reason?: string };
    template: { path: string; renderer: string };
    defaults?: { namespace?: string; protocol?: string; exposure?: string; optimizationTarget?: string; profilingMode?: string };
  };
  source?: string;
  loadedAt?: string;
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

export interface ServingApplicationCreationPlan {
  artifactId: string;
  recipe: ServingRecipe;
  model: ModelIntent;
  runtime: RuntimeIntent;
  defaults: {
    namespace: string;
    protocol: string;
    exposure: string;
    optimizationTarget: string;
    profilingMode: string;
  };
  creatable: boolean;
  message?: string;
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

export interface ObservabilitySummary {
  servingApplicationId: string;
  clusterId: string;
  namespace: string;
  prometheusUrl?: string;
  results: Array<{ name: string; description: string; query: string; value?: string; error?: string; fetchedAt: string }>;
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
  const response = await fetch(`${baseUrl}${path}`, { ...init, headers });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error || response.statusText);
  }
  return response.json() as Promise<T>;
}

export const api = {
  listClusters: () => request<InferenceCluster[]>('/v1/clusters'),
  getAcceleratorInventory: (clusterId: string) => request<AcceleratorInventory>(`/v1/clusters/${clusterId}/accelerator-inventory`),
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
  listModelArtifacts: () => request<ModelArtifact[]>('/v1/artifacts'),
  createModelArtifact: (input: CreateModelArtifactInput) => request<ModelArtifact>('/v1/artifacts', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  listRecipes: () => request<ServingRecipe[]>('/v1/recipes'),
  listServingApplicationCreationPlans: (artifactId: string) => request<ServingApplicationCreationPlan[]>(`/v1/artifacts/${artifactId}/app-plans`),
  listServingApplications: () => request<ServingApplication[]>('/v1/apps'),
  createServingApplication: (input: CreateServingApplicationInput) => request<ServingApplication>('/v1/apps', {
    method: 'POST',
    body: JSON.stringify(input),
  }),
  createPreviewTask: (appId: string) => request<Task>(`/v1/apps/${appId}/tasks/preview`, { method: 'POST' }),
  createApplyTask: (appId: string) => request<Task>(`/v1/apps/${appId}/tasks/apply`, { method: 'POST' }),
  createRedeployTask: (appId: string) => request<Task>(`/v1/apps/${appId}/tasks/redeploy`, { method: 'POST' }),
  createRetireTask: (appId: string) => request<Task>(`/v1/apps/${appId}/tasks/retire`, { method: 'POST' }),
  createDiagnosticsTask: (appId: string) => request<Task>(`/v1/apps/${appId}/tasks/diagnostics`, { method: 'POST' }),
  listServingApplicationTransitions: (appId: string) => request<ServingApplicationTransition[]>(`/v1/apps/${appId}/transitions`),
  listTasks: () => request<Task[]>('/v1/tasks'),
  listEndpoints: () => request<EndpointRegistryEntry[]>('/v1/endpoints'),
  getObservabilityEntry: (appId: string) => request<ObservabilityEntry>(`/v1/apps/${appId}/observability`),
  getObservabilitySummary: (appId: string) => request<ObservabilitySummary>(`/v1/apps/${appId}/observability/summary`),
  listAuditRecords: () => request<AuditRecord[]>('/v1/audit-records'),
};
