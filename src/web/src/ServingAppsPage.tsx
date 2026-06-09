import { FormEvent, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type CreateServingApplicationInput, type ObservabilitySummary, type ServingApplication, type ServingApplicationCreationPlan, type Task } from './api';
import { useI18n } from './i18n';
import { type ServingApplicationAction, useServingApplicationControl } from './servingApplicationControl';

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
    namespace: '',
    endpointName: 'deepseek-v4-flash',
    recipeId: '',
  });
  const [selectedAppId, setSelectedAppId] = useState('');

  const selectedArtifact = artifacts.data?.find((artifact) => artifact.id === form.artifactId);
  const creationPlans = useQuery({
    queryKey: ['serving-application-creation-plans', form.artifactId],
    queryFn: () => api.listServingApplicationCreationPlans(form.artifactId),
    enabled: Boolean(form.artifactId),
  });
  const selectedPlan = creationPlans.data?.find((plan) => plan.recipe.metadata.id === form.recipeId) ?? creationPlans.data?.[0];
  const selectedRecipe = selectedPlan?.recipe;
  const createApp = useMutation({
    mutationFn: (input: CreateServingApplicationInput) => api.createServingApplication(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['serving-applications'] });
    },
  });

  const control = useServingApplicationControl({ apps: apps.data, recipes: recipes.data, tasks: tasks.data, endpoints: endpoints.data, selectedAppId });

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

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!selectedArtifact || !selectedPlan) return;
    createApp.mutate({
      projectId: form.projectId,
      name: form.name,
      model: {
        family: selectedPlan.model.family,
        variant: selectedPlan.model.variant,
        artifactId: selectedPlan.model.artifactId,
        quantization: selectedPlan.model.quantization,
      },
      placement: {
        clusterId: form.clusterId,
        namespace: form.namespace || selectedPlan.defaults.namespace,
      },
      runtime: {
        backend: selectedPlan.runtime.backend,
        topology: selectedPlan.runtime.topology,
        recipe: selectedPlan.runtime.recipe,
      },
      service: {
        endpointName: form.endpointName,
        protocol: selectedPlan.defaults.protocol,
        exposure: selectedPlan.defaults.exposure,
      },
      optimization: {
        target: selectedPlan.defaults.optimizationTarget,
        profilingMode: selectedPlan.defaults.profilingMode,
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
          <SelectField label={t('artifact')} value={form.artifactId} onChange={(value) => setForm({ ...form, artifactId: value, recipeId: '', namespace: '' })} options={(artifacts.data ?? []).map((artifact) => [artifact.id, `${artifact.family}/${artifact.variant}:${artifact.revision}`])} />
          <SelectField label={t('recipe')} value={selectedRecipe?.metadata.id || ''} onChange={(value) => setForm({ ...form, recipeId: value })} options={(creationPlans.data ?? []).map((plan) => [plan.recipe.metadata.id, `${plan.recipe.metadata.name} (${plan.recipe.spec.support.status})`])} />
          {selectedPlan ? <RecipeWarning plan={selectedPlan} /> : <p className="muted">No matching recipe for selected artifact.</p>}
          <InputField label={t('name')} value={form.name} onChange={(value) => setForm({ ...form, name: value })} />
          <InputField label={t('namespace')} value={form.namespace || selectedPlan?.defaults.namespace || ''} onChange={(value) => setForm({ ...form, namespace: value })} />
          <InputField label={t('endpointName')} value={form.endpointName} onChange={(value) => setForm({ ...form, endpointName: value })} />
          <div className="grid">
            <span className="badge muted">{t('backend')}: {selectedRecipe?.spec.runtime.backend || '-'}</span>
            <span className="badge muted">{t('topology')}: {selectedRecipe?.spec.runtime.topology || '-'}</span>
            <span className="badge muted">{t('recipe')}: {selectedRecipe?.metadata.id || '-'}</span>
          </div>
          <button className="button" disabled={createApp.isPending || !form.projectId || !form.clusterId || !form.artifactId || !selectedPlan?.creatable}>{t('createServingApp')}</button>
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
                  <ServingAppRow key={app.id} app={app} endpoint={control.endpointsByApp.get(app.id)} selected={selectedAppId === app.id} onSelect={() => setSelectedAppId(app.id)} onAction={(action) => control.actionMutation.mutate({ appId: app.id, action })} disabled={control.actionMutation.isPending} />
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
              <DiagnosticsTask task={control.latestDiagnosticsTask} />
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

function ServingAppRow({ app, endpoint, selected, onSelect, onAction, disabled }: { app: ServingApplication; endpoint?: string; selected: boolean; onSelect: () => void; onAction: (action: ServingApplicationAction) => void; disabled: boolean }) {
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

function RecipeWarning({ plan }: { plan: ServingApplicationCreationPlan }) {
  const status = plan.recipe.spec.support.status;
  if (status === 'supported') {
    return <p className="muted">Recipe support: supported.</p>;
  }
  const message = plan.message || 'No support note provided.';
  return <p className={plan.creatable ? 'muted' : 'error'}>Recipe support: {status}. {message}</p>;
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
