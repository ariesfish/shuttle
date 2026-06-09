import { FormEvent, useState } from 'react';
import { type CreateServingApplicationInput, type InferenceCluster, type ModelArtifact, type Project, type ServingApplicationCreationPlan } from './api';
import { useI18n } from './i18n';

export function ServingAppCreateForm({ projects, clusters, artifacts, creationPlans, creating, createError, onArtifactChange, onCreate }: { projects: Project[]; clusters: InferenceCluster[]; artifacts: ModelArtifact[]; creationPlans?: ServingApplicationCreationPlan[]; creating: boolean; createError?: string; onArtifactChange: (artifactId: string) => void; onCreate: (input: CreateServingApplicationInput) => void }) {
  const { t } = useI18n();
  const [form, setForm] = useState({
    projectId: '',
    clusterId: '',
    artifactId: '',
    name: 'DeepSeek V4 Flash',
    namespace: '',
    endpointName: 'deepseek-v4-flash',
    recipeId: '',
  });
  const selectedPlan = creationPlans?.find((plan) => plan.recipe.metadata.id === form.recipeId) ?? creationPlans?.[0];
  const selectedRecipe = selectedPlan?.recipe;

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!selectedPlan) return;
    onCreate({
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

  function setArtifactId(artifactId: string) {
    setForm({ ...form, artifactId, recipeId: '', namespace: '' });
    onArtifactChange(artifactId);
  }

  return (
    <form className="form" onSubmit={submit}>
      <SelectField label={t('project')} value={form.projectId} onChange={(value) => setForm({ ...form, projectId: value })} options={projects.map((project) => [project.id, project.name])} />
      <SelectField label={t('cluster')} value={form.clusterId} onChange={(value) => setForm({ ...form, clusterId: value })} options={clusters.map((cluster) => [cluster.id, cluster.name])} />
      <SelectField label={t('artifact')} value={form.artifactId} onChange={setArtifactId} options={artifacts.map((artifact) => [artifact.id, `${artifact.family}/${artifact.variant}:${artifact.revision}`])} />
      <SelectField label={t('recipe')} value={selectedRecipe?.metadata.id || ''} onChange={(value) => setForm({ ...form, recipeId: value })} options={(creationPlans ?? []).map((plan) => [plan.recipe.metadata.id, `${plan.recipe.metadata.name} (${plan.recipe.spec.support.status})`])} />
      {selectedPlan ? <RecipeWarning plan={selectedPlan} /> : <p className="muted">No matching recipe for selected artifact.</p>}
      <InputField label={t('name')} value={form.name} onChange={(value) => setForm({ ...form, name: value })} />
      <InputField label={t('namespace')} value={form.namespace || selectedPlan?.defaults.namespace || ''} onChange={(value) => setForm({ ...form, namespace: value })} />
      <InputField label={t('endpointName')} value={form.endpointName} onChange={(value) => setForm({ ...form, endpointName: value })} />
      <div className="grid">
        <span className="badge muted">{t('backend')}: {selectedRecipe?.spec.runtime.backend || '-'}</span>
        <span className="badge muted">{t('topology')}: {selectedRecipe?.spec.runtime.topology || '-'}</span>
        <span className="badge muted">{t('recipe')}: {selectedRecipe?.metadata.id || '-'}</span>
      </div>
      <button className="button" disabled={creating || !form.projectId || !form.clusterId || !form.artifactId || !selectedPlan?.creatable}>{t('createServingApp')}</button>
      {createError ? <p className="error">{createError}</p> : null}
    </form>
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
