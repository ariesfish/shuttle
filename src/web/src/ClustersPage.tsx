import { FormEvent, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type ClusterAgent, type CreateClusterInput, type InferenceCluster } from './api';
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

  const agentsByCluster = useMemo(() => {
    const map = new Map<string, ClusterAgent>();
    for (const agent of agents.data ?? []) {
      map.set(agent.clusterId, agent);
    }
    return map;
  }, [agents.data]);

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
                <th>{t('installCommand')}</th>
              </tr>
            </thead>
            <tbody>
              {clusters.data.map((cluster) => (
                <ClusterRow key={cluster.id} cluster={cluster} agent={agentsByCluster.get(cluster.id)} />
              ))}
            </tbody>
          </table>
        ) : clusters.isLoading ? null : <p className="muted">{t('noData')}</p>}
      </section>
    </div>
  );
}

function ClusterRow({ cluster, agent }: { cluster: InferenceCluster; agent?: ClusterAgent }) {
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
  return `cd src\ngo run ./cmd/cluster-agent \\\n  -management-url ${baseUrl} \\\n  -cluster-id ${clusterId} \\\n  -auth-token ${token || '<token>'} \\\n  -capability dynamo=true,backend=vllm`;
}

function formatCapabilities(capabilities?: Record<string, string>) {
  if (!capabilities || Object.keys(capabilities).length === 0) {
    return '-';
  }
  return Object.entries(capabilities).map(([key, value]) => `${key}=${value}`).join(', ');
}

function formatDate(value: string | undefined, fallback: string) {
  if (!value) {
    return fallback;
  }
  return new Date(value).toLocaleString();
}
