import { FormEvent, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type AcceleratorPoolSummary, type CreateAcceleratorPoolInput } from './api';
import { useI18n } from './i18n';

export function AcceleratorPoolsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const clusters = useQuery({ queryKey: ['clusters'], queryFn: api.listClusters });
  const summaries = useQuery({ queryKey: ['acceleratorPoolSummaries'], queryFn: () => api.listAcceleratorPoolSummaries(), refetchInterval: 5000 });
  const [selectorText, setSelectorText] = useState('');
  const [form, setForm] = useState<CreateAcceleratorPoolInput>({ clusterId: '', name: '', description: '', nodeSelector: {} });
  const createPool = useMutation({
    mutationFn: api.createAcceleratorPool,
    onSuccess: () => {
      setForm({ clusterId: '', name: '', description: '', nodeSelector: {} });
      setSelectorText('');
      queryClient.invalidateQueries({ queryKey: ['acceleratorPoolSummaries'] });
    },
  });
  const clusterName = useMemo(() => new Map((clusters.data ?? []).map((cluster) => [cluster.id, cluster.name])), [clusters.data]);

  function submit(event: FormEvent) {
    event.preventDefault();
    createPool.mutate({ ...form, nodeSelector: parseSelector(selectorText) });
  }

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('acceleratorPoolsTitle')}</h1>
        <p className="page-description">{t('acceleratorPoolsDescription')}</p>
        <form className="form" onSubmit={submit}>
          <label className="field">
            <span className="label">{t('cluster')}</span>
            <select className="select" required value={form.clusterId} onChange={(event) => setForm({ ...form, clusterId: event.target.value })}>
              <option value="">-</option>
              {(clusters.data ?? []).map((cluster) => <option key={cluster.id} value={cluster.id}>{cluster.name} ({cluster.id})</option>)}
            </select>
          </label>
          <label className="field">
            <span className="label">{t('name')}</span>
            <input className="input" required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t('description')}</span>
            <input className="input" value={form.description ?? ''} onChange={(event) => setForm({ ...form, description: event.target.value })} />
          </label>
          <label className="field">
            <span className="label">{t('nodeSelector')}</span>
            <input className="input" placeholder="key=value,key2=value2" value={selectorText} onChange={(event) => setSelectorText(event.target.value)} />
          </label>
          <button className="button" disabled={createPool.isPending}>{t('create')}</button>
          {createPool.error ? <p className="error">{createPool.error.message}</p> : null}
        </form>
      </section>
      <section className="card">
        {summaries.isLoading ? <p>{t('loading')}</p> : null}
        {summaries.error ? <p className="error">{summaries.error.message}</p> : null}
        {summaries.data?.length ? (
          <table className="table">
            <thead><tr><th>{t('name')}</th><th>{t('cluster')}</th><th>{t('summary')}</th><th>{t('warnings')}</th></tr></thead>
            <tbody>{summaries.data.map((summary) => <PoolRow key={summary.pool.id} summary={summary} clusterName={clusterName.get(summary.pool.clusterId)} />)}</tbody>
          </table>
        ) : summaries.isLoading ? null : <p className="muted">{t('noData')}</p>}
      </section>
    </div>
  );
}

function PoolRow({ summary, clusterName }: { summary: AcceleratorPoolSummary; clusterName?: string }) {
  const { t } = useI18n();
  return (
    <tr>
      <td><strong>{summary.pool.name}</strong><div className="muted">{formatSelector(summary.pool.nodeSelector)}</div></td>
      <td>{clusterName || summary.pool.clusterId}<div className="muted"><code>{summary.pool.clusterId}</code></div></td>
      <td>
        <span className={`badge ${summary.freshness === 'fresh' ? '' : 'muted'}`}>{summary.freshness}</span>
        <div className="muted">{t('nodeCount')}: {summary.nodeCount}</div>
        <div className="muted">{t('acceleratorCount')}: {summary.acceleratorCount}</div>
        <div className="muted">{t('acceleratorModels')}: {formatRecord(summary.acceleratorModels)}</div>
        <div className="muted">{t('memoryMiB')}: {formatRecord(summary.memoryMiBSummary)}</div>
      </td>
      <td>{summary.warnings?.length ? summary.warnings.join(', ') : '-'}</td>
    </tr>
  );
}

function parseSelector(value: string) {
  const selector: Record<string, string> = {};
  for (const item of value.split(',')) {
    const [key, ...rest] = item.split('=');
    if (key?.trim()) {
      selector[key.trim()] = rest.join('=').trim();
    }
  }
  return selector;
}

function formatSelector(selector?: Record<string, string>) {
  if (!selector || Object.keys(selector).length === 0) {
    return '-';
  }
  return Object.entries(selector).map(([key, value]) => `${key}=${value}`).join(', ');
}

function formatRecord(record?: Record<string, number>) {
  if (!record || Object.keys(record).length === 0) {
    return '-';
  }
  return Object.entries(record).map(([key, value]) => `${key}=${value}`).join(', ');
}
