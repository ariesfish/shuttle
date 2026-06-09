import { FormEvent, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type CreateModelArtifactInput } from './api';
import { useI18n } from './i18n';

const defaultArtifact: CreateModelArtifactInput = {
  family: 'deepseek-v4',
  variant: 'flash',
  revision: '',
  pvcName: '',
  pvcMountPath: '/home/dynamo/.cache/huggingface',
  pvcModelPath: '',
  hostCachePath: '/data/cache/hub',
  quantization: 'fp8',
};

export function ArtifactsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateModelArtifactInput>(defaultArtifact);
  const artifacts = useQuery({ queryKey: ['artifacts'], queryFn: api.listModelArtifacts });
  const createArtifact = useMutation({
    mutationFn: api.createModelArtifact,
    onSuccess: () => {
      setForm(defaultArtifact);
      queryClient.invalidateQueries({ queryKey: ['artifacts'] });
    },
  });

  function submit(event: FormEvent) {
    event.preventDefault();
    createArtifact.mutate(form);
  }

  function update<K extends keyof CreateModelArtifactInput>(key: K, value: CreateModelArtifactInput[K]) {
    setForm({ ...form, [key]: value });
  }

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('artifactsTitle')}</h1>
        <p className="page-description">{t('artifactsDescription')}</p>
        <form className="form" onSubmit={submit}>
          <Field label={t('family')} value={form.family} onChange={(value) => update('family', value)} required />
          <Field label={t('variant')} value={form.variant} onChange={(value) => update('variant', value)} required />
          <Field label={t('revision')} value={form.revision} onChange={(value) => update('revision', value)} required />
          <Field label={t('pvcName')} value={form.pvcName ?? ''} onChange={(value) => update('pvcName', value)} />
          <Field label={t('pvcMountPath')} value={form.pvcMountPath} onChange={(value) => update('pvcMountPath', value)} required />
          <Field label={t('pvcModelPath')} value={form.pvcModelPath} onChange={(value) => update('pvcModelPath', value)} required />
          <Field label={t('hostCachePath')} value={form.hostCachePath ?? ''} onChange={(value) => update('hostCachePath', value)} />
          <Field label={t('quantization')} value={form.quantization} onChange={(value) => update('quantization', value)} required />
          <div className="toolbar">
            <button className="button" disabled={createArtifact.isPending}>{t('createArtifact')}</button>
            <button className="button secondary" type="button" onClick={() => setForm(defaultArtifact)}>{t('reset')}</button>
          </div>
          {createArtifact.error ? <p className="error">{createArtifact.error.message}</p> : null}
        </form>
      </section>
      <section className="card">
        {artifacts.isLoading ? <p>{t('loading')}</p> : null}
        {artifacts.error ? <p className="error">{t('error')}: {artifacts.error.message}</p> : null}
        {artifacts.data?.length ? (
          <table className="table">
            <thead>
              <tr>
                <th>{t('family')}</th>
                <th>{t('revision')}</th>
                <th>{t('pvcModelPath')}</th>
                <th>{t('quantization')}</th>
              </tr>
            </thead>
            <tbody>
              {artifacts.data.map((artifact) => (
                <tr key={artifact.id}>
                  <td><strong>{artifact.family}/{artifact.variant}</strong><div className="muted"><code>{artifact.id}</code></div></td>
                  <td>{artifact.revision}</td>
                  <td><code>{artifact.pvcMountPath}/{artifact.pvcModelPath}</code></td>
                  <td><span className="badge">{artifact.quantization}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : artifacts.isLoading ? null : <p className="muted">{t('noData')}</p>}
      </section>
    </div>
  );
}

function Field({ label, value, onChange, required = false }: { label: string; value: string; onChange: (value: string) => void; required?: boolean }) {
  return (
    <label className="field">
      <span className="label">{label}</span>
      <input className="input" required={required} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}
