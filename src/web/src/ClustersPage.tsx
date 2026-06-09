import { FormEvent, useMemo, useState } from 'react';
import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type AcceleratorInventory, type ClusterAgent, type CreateClusterInput, type InferenceCluster } from './api';
import { getApiSettings } from './api';
import { useI18n } from './i18n';

export function ClustersPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateClusterInput>({ name: '', description: '', prometheusUrl: '', grafanaUrl: '' });
  const clusters = useQuery({ queryKey: ['clusters'], queryFn: api.listClusters, refetchInterval: 5000 });
  const agents = useQuery({ queryKey: ['agents'], queryFn: api.listAgents, refetchInterval: 2000 });
  const createCluster = useMutation({
    mutationFn: api.createCluster,
    onSuccess: () => {
      setForm({ name: '', description: '', prometheusUrl: '', grafanaUrl: '' });
      queryClient.invalidateQueries({ queryKey: ['clusters'] });
    },
  });

  const inventoryQueries = useQueries({
    queries: (clusters.data ?? []).map((cluster) => ({
      queryKey: ['acceleratorInventory', cluster.id],
      queryFn: () => api.getAcceleratorInventory(cluster.id),
      refetchInterval: 5000,
    })),
  });
  const agentsByCluster = useMemo(() => {
    const map = new Map<string, ClusterAgent>();
    for (const agent of agents.data ?? []) {
      map.set(agent.clusterId, agent);
    }
    return map;
  }, [agents.data]);
  const inventoryByCluster = useMemo(() => {
    const map = new Map<string, AcceleratorInventory>();
    (clusters.data ?? []).forEach((cluster, index) => {
      const inventory = inventoryQueries[index]?.data;
      if (inventory) {
        map.set(cluster.id, inventory);
      }
    });
    return map;
  }, [clusters.data, inventoryQueries]);

  function submit(event: FormEvent) {
    event.preventDefault();
    createCluster.mutate(form);
  }

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('clustersTitle')}</h1>
        <p className="page-description">{t('clustersDescription')}</p>
        <form className="form" onSubmit={submit}>
          <label className="field">
            <span className="label">{t('name')}</span>
            <input className="input" required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t('description')}</span>
            <input className="input" value={form.description ?? ''} onChange={(event) => setForm({ ...form, description: event.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t('prometheusUrl')}</span>
            <input className="input" value={form.prometheusUrl ?? ''} onChange={(event) => setForm({ ...form, prometheusUrl: event.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t('grafanaUrl')}</span>
            <input className="input" value={form.grafanaUrl ?? ''} onChange={(event) => setForm({ ...form, grafanaUrl: event.target.value })} />
          </label>
          <div className="toolbar">
            <button className="button" disabled={createCluster.isPending}>{t('create')}</button>
            <button className="button secondary" type="button" onClick={() => setForm({ name: '', description: '', prometheusUrl: '', grafanaUrl: '' })}>{t('reset')}</button>
          </div>
          {createCluster.error ? <p className="error">{createCluster.error.message}</p> : null}
        </form>
      </section>
      <section className="card">
        {clusters.isLoading || agents.isLoading ? <p>{t('loading')}</p> : null}
        {clusters.error ? <p className="error">{t('error')}: {clusters.error.message}</p> : null}
        {clusters.data?.length ? (
          <table className="table">
            <thead>
              <tr>
                <th>{t('name')}</th>
                <th>{t('clusterId')}</th>
                <th>{t('agentStatus')}</th>
                <th>{t('acceleratorInventory')}</th>
                <th>{t('installCommand')}</th>
              </tr>
            </thead>
            <tbody>
              {clusters.data.map((cluster) => (
                <ClusterRow key={cluster.id} cluster={cluster} agent={agentsByCluster.get(cluster.id)} inventory={inventoryByCluster.get(cluster.id)} />
              ))}
            </tbody>
          </table>
        ) : clusters.isLoading ? null : <p className="muted">{t('noData')}</p>}
      </section>
    </div>
  );
}

function ClusterRow({ cluster, agent, inventory }: { cluster: InferenceCluster; agent?: ClusterAgent; inventory?: AcceleratorInventory }) {
  const { t } = useI18n();
  const command = agentCommand(cluster.id);
  return (
    <tr>
      <td>
        <strong>{cluster.name}</strong>
        <div className="muted">{cluster.description}</div>
        <div className="muted">{cluster.prometheusUrl}</div>
        <div className="muted">{cluster.grafanaUrl}</div>
      </td>
      <td><code>{cluster.id}</code></td>
      <td>
        {agent ? <span className="badge">{t('online')}</span> : <span className="badge muted">{t('unknown')}</span>}
        <div className="muted">{t('version')}: {agent?.version || t('unknown')}</div>
        <div className="muted">{t('lastHeartbeat')}: {formatDate(agent?.lastHeartbeat, t('never'))}</div>
        <div className="muted">{t('capabilities')}: {formatCapabilities(agent?.capabilities)}</div>
        <div className="muted">{t('inventoryRevision')}: {agent?.lastInventoryRevision || '-'}</div>
      </td>
      <td>
        <span className={`badge ${inventory?.freshness === 'fresh' ? '' : 'muted'}`}>{inventory?.freshness || agent?.lastInventoryFreshness || t('missing')}</span>
        <div className="muted">{t('nodeCount')}: {inventory?.nodes?.length ?? 0}</div>
        <div className="muted">{t('nodeNames')}: {formatNodeNames(inventory)}</div>
        <div className="muted">{t('acceleratorResources')}: {formatAcceleratorResources(inventory)}</div>
        <div className="muted">{t('nvidiaAccelerators')}: {formatNvidiaAccelerators(inventory)}</div>
        <div className="muted">{t('connectivity')}: {formatConnectivity(inventory)}</div>
        <div className="muted">{t('observedAt')}: {formatDate(inventory?.observedAt || agent?.lastInventoryObservedAt, t('never'))}</div>
        <div className="muted">{t('probeStatus')}: {formatProbeStatuses(inventory?.probeStatuses)}</div>
      </td>
      <td>
        <p className="muted">{t('copyHint')}</p>
        <pre className="code">{command}</pre>
      </td>
    </tr>
  );
}

function agentCommand(clusterId: string) {
  const { baseUrl, token } = getApiSettings();
  return `cd src\ngo run ./cmd/cluster-agent \\\n  -management-url ${baseUrl} \\\n  -cluster-id ${clusterId} \\\n  -auth-token ${token || '<token>'} \\\n  -executor-mode fake \\\n  -capability dynamo=true,backend=vllm`;
}

function formatCapabilities(capabilities?: Record<string, string>) {
  if (!capabilities || Object.keys(capabilities).length === 0) {
    return '-';
  }
  return Object.entries(capabilities).map(([key, value]) => `${key}=${value}`).join(', ');
}

function formatNodeNames(inventory?: AcceleratorInventory) {
  const names = inventory?.nodes?.map((node) => node.name).filter(Boolean) ?? [];
  return names.length ? names.join(', ') : '-';
}

function formatAcceleratorResources(inventory?: AcceleratorInventory) {
  const resources = new Set<string>();
  for (const node of inventory?.nodes ?? []) {
    for (const resource of node.acceleratorResourceNames ?? []) {
      resources.add(resource);
    }
  }
  return resources.size ? Array.from(resources).sort().join(', ') : '-';
}

function formatNvidiaAccelerators(inventory?: AcceleratorInventory) {
  const summaries: string[] = [];
  for (const node of inventory?.nodes ?? []) {
    for (const accelerator of node.accelerators ?? []) {
      if (accelerator.vendor !== 'nvidia') {
        continue;
      }
      const product = accelerator.product || 'unknown NVIDIA';
      const count = accelerator.deviceCount ? ` x${accelerator.deviceCount}` : '';
      const memory = accelerator.memoryMiB ? ` ${accelerator.memoryMiB}MiB` : '';
      summaries.push(`${product}${count}${memory}`);
    }
  }
  return summaries.length ? summaries.join(', ') : '-';
}

function formatConnectivity(inventory?: AcceleratorInventory) {
  const facts: string[] = [];
  for (const node of inventory?.nodes ?? []) {
    for (const fact of node.connectivity ?? []) {
      facts.push(`${fact.type}=${fact.present ? 'present' : 'missing'}(${fact.confidence})`);
    }
  }
  return facts.length ? Array.from(new Set(facts)).sort().join(', ') : '-';
}

function formatProbeStatuses(probes?: Array<{ name: string; status: string; message?: string }>) {
  if (!probes || probes.length === 0) {
    return '-';
  }
  return probes.map((probe) => `${probe.name}=${probe.status}`).join(', ');
}

function formatDate(value: string | undefined, fallback: string) {
  if (!value) {
    return fallback;
  }
  return new Date(value).toLocaleString();
}
