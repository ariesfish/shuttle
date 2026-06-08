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
};
