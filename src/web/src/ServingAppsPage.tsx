import { FormEvent, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type CreateServingApplicationInput, type ObservabilitySummary, type ServingApplication, type ServingRecipe, type Task } from './api';
import { useI18n } from './i18n';

const fallbackDefaults = {
  namespace: 'dynamo-system',
  protocol: 'openai-compatible',
  exposure: 'cluster-local',
  optimizationTarget: 'throughput',
  profilingMode: 'disabled',
};

export function ServingAppsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects });
  const clusters = useQuery({ queryKey: ['clusters'], queryFn: api.listClusters });
  const artifacts = useQuery({ queryKey: ['model-artifacts'], queryFn: api.listModelArtifacts });
  const recipes = useQuery({ queryKey: ['recipes'], queryFn: api.listRecipes });
  const apps = useQuery({ queryKey: ['serving-applications'], queryFn: api.listServingApplications, refetchInterval: 2000 });
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: api.listTasks, refetchInterval: 2000 });
  const endpoints = useQuery({ queryKey: ['endpoints'], queryFn: api.listEndpoints, refetchInterval: 2000 });

  const [form, setForm] = useState({
    projectId: '',
    clusterId: '',
    artifactId: '',
    name: 'DeepSeek V4 Flash',
    namespace: fallbackDefaults.namespace,
    endpointName: 'deepseek-v4-flash',
    recipeId: '',
  });
  const [selectedAppId, setSelectedAppId] = useState('');

  const selectedArtifact = artifacts.data?.find((artifact) => artifact.id === form.artifactId);
  const matchingRecipes = useMemo(() => {
    if (!selectedArtifact) return recipes.data ?? [];
    return (recipes.data ?? []).filter((recipe) => recipe.spec.model.family === selectedArtifact.family && recipe.spec.model.variants.includes(selectedArtifact.variant) && recipe.spec.model.quantizations.includes(selectedArtifact.quantization));
  }, [recipes.data, selectedArtifact]);
  const selectedRecipe = matchingRecipes.find((recipe) => recipe.metadata.id === form.recipeId) ?? matchingRecipes[0];
  const createApp = useMutation({
    mutationFn: (input: CreateServingApplicationInput) => api.createServingApplication(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['serving-applications'] });
    },
  });

  const actionMutation = useMutation({
    mutationFn: async ({ appId, action }: { appId: string; action: 'preview' | 'apply' | 'redeploy' | 'retire' | 'diagnostics' }) => {
      const app = apps.data?.find((candidate) => candidate.id === appId);
      const recipe = recipeForApp(recipes.data ?? [], app);
      if ((action === 'apply' || action === 'redeploy') && recipe?.spec.support.status === 'experimental' && !confirm(`Recipe is experimental: ${recipe.spec.support.warning || recipe.metadata.name}`)) {
        throw new Error('action cancelled');
      }
      if (action === 'preview') return api.createPreviewTask(appId);
      if (action === 'apply') return api.createApplyTask(appId);
      if (action === 'redeploy') return api.createRedeployTask(appId);
      if (action === 'diagnostics') return api.createDiagnosticsTask(appId);
      return api.createRetireTask(appId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['serving-applications'] });
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });

  const transitions = useQuery({
    queryKey: ['serving-application-transitions', selectedAppId],
    queryFn: () => api.listServingApplicationTransitions(selectedAppId),
    enabled: Boolean(selectedAppId),
    refetchInterval: 2000,
  });
  const observabilitySummary = useQuery({
    queryKey: ['observability-summary', selectedAppId],
    queryFn: () => api.getObservabilitySummary(selectedAppId),
    enabled: Boolean(selectedAppId),
    refetchInterval: 10000,
  });

  const endpointsByApp = useMemo(() => {
    const map = new Map<string, string>();
    for (const endpoint of endpoints.data ?? []) {
      map.set(endpoint.servingApplicationId, endpoint.url);
    }
    return map;
  }, [endpoints.data]);

  const latestDiagnosticsTask = useMemo(() => {
    return [...(tasks.data ?? [])]
      .reverse()
      .find((task) => task.type === 'FetchDiagnostics' && task.payload?.servingApplicationId === selectedAppId);
  }, [selectedAppId, tasks.data]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!selectedArtifact || !selectedRecipe) return;
    createApp.mutate({
      projectId: form.projectId,
      name: form.name,
      model: {
        family: selectedArtifact.family,
        variant: selectedArtifact.variant,
        artifactId: selectedArtifact.id,
        quantization: selectedArtifact.quantization,
      },
      placement: {
        clusterId: form.clusterId,
        namespace: form.namespace,
      },
      runtime: {
        backend: selectedRecipe.spec.runtime.backend,
        topology: selectedRecipe.spec.runtime.topology,
        recipe: selectedRecipe.metadata.id,
      },
      service: {
        endpointName: form.endpointName,
        protocol: selectedRecipe.spec.defaults?.protocol || fallbackDefaults.protocol,
        exposure: selectedRecipe.spec.defaults?.exposure || fallbackDefaults.exposure,
      },
      optimization: {
        target: selectedRecipe.spec.defaults?.optimizationTarget || fallbackDefaults.optimizationTarget,
        profilingMode: selectedRecipe.spec.defaults?.profilingMode || fallbackDefaults.profilingMode,
      },
    });
  }

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('servingAppsTitle')}</h1>
        <p className="page-description">{t('servingAppsDescription')}</p>
        <form className="form" onSubmit={submit}>
          <SelectField label={t('project')} value={form.projectId} onChange={(value) => setForm({ ...form, projectId: value })} options={(projects.data ?? []).map((project) => [project.id, project.name])} />
          <SelectField label={t('cluster')} value={form.clusterId} onChange={(value) => setForm({ ...form, clusterId: value })} options={(clusters.data ?? []).map((cluster) => [cluster.id, cluster.name])} />
          <SelectField label={t('artifact')} value={form.artifactId} onChange={(value) => setForm({ ...form, artifactId: value, recipeId: '' })} options={(artifacts.data ?? []).map((artifact) => [artifact.id, `${artifact.family}/${artifact.variant}:${artifact.revision}`])} />
          <SelectField label={t('recipe')} value={selectedRecipe?.metadata.id || ''} onChange={(value) => setForm({ ...form, recipeId: value })} options={matchingRecipes.map((recipe) => [recipe.metadata.id, `${recipe.metadata.name} (${recipe.spec.support.status})`])} />
          {selectedRecipe ? <RecipeWarning recipe={selectedRecipe} /> : <p className="muted">No matching recipe for selected artifact.</p>}
          <InputField label={t('name')} value={form.name} onChange={(value) => setForm({ ...form, name: value })} />
          <InputField label={t('namespace')} value={form.namespace} onChange={(value) => setForm({ ...form, namespace: value })} />
          <InputField label={t('endpointName')} value={form.endpointName} onChange={(value) => setForm({ ...form, endpointName: value })} />
          <div className="grid">
            <span className="badge muted">{t('backend')}: {selectedRecipe?.spec.runtime.backend || '-'}</span>
            <span className="badge muted">{t('topology')}: {selectedRecipe?.spec.runtime.topology || '-'}</span>
            <span className="badge muted">{t('recipe')}: {selectedRecipe?.metadata.id || '-'}</span>
          </div>
          <button className="button" disabled={createApp.isPending || !form.projectId || !form.clusterId || !form.artifactId || !selectedRecipe || selectedRecipe.spec.support.status === 'blocked'}>{t('createServingApp')}</button>
          {createApp.error ? <p className="error">{createApp.error.message}</p> : null}
        </form>
      </section>
      <section className="grid">
        <section className="card">
          {apps.isLoading ? <p>{t('loading')}</p> : null}
          {apps.error ? <p className="error">{t('error')}: {apps.error.message}</p> : null}
          {apps.data?.length ? (
            <table className="table">
              <thead>
                <tr>
                  <th>{t('name')}</th>
                  <th>{t('phase')}</th>
                  <th>{t('endpoint')}</th>
                  <th>{t('actions')}</th>
                </tr>
              </thead>
              <tbody>
                {apps.data.map((app) => (
                  <ServingAppRow key={app.id} app={app} endpoint={endpointsByApp.get(app.id)} selected={selectedAppId === app.id} onSelect={() => setSelectedAppId(app.id)} onAction={(action) => actionMutation.mutate({ appId: app.id, action })} disabled={actionMutation.isPending} />
                ))}
              </tbody>
            </table>
          ) : apps.isLoading ? null : <p className="muted">{t('noData')}</p>}
        </section>
        <section className="card">
          <h2>History & Diagnostics</h2>
          {!selectedAppId ? <p className="muted">Select a Serving Application to inspect transitions and diagnostics.</p> : null}
          {selectedAppId ? (
            <div className="grid">
              <ObservabilitySummaryCard summary={observabilitySummary.data} error={observabilitySummary.error?.message} />
              <div>
                <h3>Transitions</h3>
                {transitions.data?.slice(-8).reverse().map((transition) => (
                  <div key={transition.id} className="card" style={{ marginBottom: 8 }}>
                    <strong>{transition.from || '-'} → {transition.to}</strong> <span className="badge muted">{transition.actor}</span>
                    <div className="muted"><code>{transition.taskId || transition.id}</code></div>
                    {transition.reason ? <div>{transition.reason}</div> : null}
                  </div>
                )) ?? <p className="muted">{t('noData')}</p>}
              </div>
              <DiagnosticsTask task={latestDiagnosticsTask} />
            </div>
          ) : null}
        </section>
        <section className="card">
          <h2>{t('tasksTitle')}</h2>
          {tasks.data?.slice(-8).reverse().map((task) => (
            <div key={task.id} className="card" style={{ marginBottom: 8 }}>
              <strong>{task.type}</strong> <span className="badge muted">{task.status}</span>
              <div className="muted"><code>{task.id}</code></div>
              {task.error ? <div className="error">{task.error}</div> : null}
            </div>
          )) ?? <p className="muted">{t('noData')}</p>}
        </section>
      </section>
    </div>
  );
}

function ServingAppRow({ app, endpoint, selected, onSelect, onAction, disabled }: { app: ServingApplication; endpoint?: string; selected: boolean; onSelect: () => void; onAction: (action: 'preview' | 'apply' | 'redeploy' | 'retire' | 'diagnostics') => void; disabled: boolean }) {
  const { t } = useI18n();
  const url = endpoint || app.endpointUrl;
  return (
    <tr>
      <td><strong>{app.name}</strong>{selected ? <span className="badge muted" style={{ marginLeft: 8 }}>selected</span> : null}<div className="muted"><code>{app.id}</code></div><div className="muted">{app.model.family}/{app.model.variant}</div></td>
      <td><span className="badge">{app.phase}</span></td>
      <td>{url ? <code>{url}</code> : <span className="muted">-</span>}</td>
      <td>
        <div className="toolbar">
          <button className="button secondary" disabled={disabled} onClick={onSelect}>Inspect</button>
          <button className="button secondary" disabled={disabled} onClick={() => onAction('preview')}>{t('preview')}</button>
          <button className="button secondary" disabled={disabled} onClick={() => onAction('apply')}>{t('apply')}</button>
          <button className="button secondary" disabled={disabled} onClick={() => onAction('redeploy')}>{t('redeploy')}</button>
          <button className="button secondary" disabled={disabled} onClick={() => onAction('diagnostics')}>Diagnostics</button>
          <button className="button secondary" disabled={disabled} onClick={() => onAction('retire')}>{t('retire')}</button>
        </div>
      </td>
    </tr>
  );
}

function ObservabilitySummaryCard({ summary, error }: { summary?: ObservabilitySummary; error?: string }) {
  return (
    <div>
      <h3>Observability</h3>
      {error ? <p className="error">{error}</p> : null}
      {!summary ? <p className="muted">Loading Prometheus summary...</p> : null}
      {summary?.results.map((result) => (
        <div key={result.name} className="card" style={{ marginBottom: 8 }}>
          <strong>{result.name}</strong> <span className="badge muted">{result.value || '-'}</span>
          <div className="muted">{result.description}</div>
          {result.error ? <div className="error">{result.error}</div> : null}
        </div>
      ))}
    </div>
  );
}

function RecipeWarning({ recipe }: { recipe: ServingRecipe }) {
  if (recipe.spec.support.status === 'supported') {
    return <p className="muted">Recipe support: supported.</p>;
  }
  const message = recipe.spec.support.warning || recipe.spec.support.reason || 'No support note provided.';
  return <p className={recipe.spec.support.status === 'blocked' ? 'error' : 'muted'}>Recipe support: {recipe.spec.support.status}. {message}</p>;
}

function recipeForApp(recipes: ServingRecipe[], app?: ServingApplication) {
  if (!app) return undefined;
  return recipes.find((recipe) => recipe.metadata.id === app.runtime.recipe);
}

function DiagnosticsTask({ task }: { task?: Task }) {
  const sections = task?.result?.sections as Array<{ name: string; output?: string; error?: string }> | undefined;
  return (
    <div>
      <h3>Diagnostics</h3>
      {!task ? <p className="muted">Run Diagnostics to fetch bounded cluster state, events, and logs.</p> : null}
      {task ? <p><span className="badge muted">{task.status}</span> <code>{task.id}</code></p> : null}
      {task?.error ? <p className="error">{task.error}</p> : null}
      {sections?.map((section) => (
        <details key={section.name} className="card" style={{ marginBottom: 8 }}>
          <summary><strong>{section.name}</strong>{section.error ? <span className="error"> — {section.error}</span> : null}</summary>
          <pre style={{ whiteSpace: 'pre-wrap', maxHeight: 240, overflow: 'auto' }}>{section.output || '-'}</pre>
        </details>
      ))}
    </div>
  );
}

function InputField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="field">
      <span className="label">{label}</span>
      <input className="input" required value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function SelectField({ label, value, onChange, options }: { label: string; value: string; onChange: (value: string) => void; options: Array<[string, string]> }) {
  return (
    <label className="field">
      <span className="label">{label}</span>
      <select className="select" required value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">-</option>
        {options.map(([optionValue, label]) => <option key={optionValue} value={optionValue}>{label}</option>)}
      </select>
    </label>
  );
}
