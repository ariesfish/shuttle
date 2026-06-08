import { FormEvent, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type CreateProjectInput } from './api';
import { useI18n } from './i18n';

export function ProjectsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<CreateProjectInput>({ name: '' });
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects });
  const createProject = useMutation({
    mutationFn: api.createProject,
    onSuccess: () => {
      setForm({ name: '' });
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    },
  });

  function submit(event: FormEvent) {
    event.preventDefault();
    createProject.mutate(form);
  }

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('projectsTitle')}</h1>
        <p className="page-description">{t('projectsDescription')}</p>
        <form className="form" onSubmit={submit}>
          <label className="field">
            <span className="label">{t('name')}</span>
            <input className="input" required value={form.name} onChange={(event) => setForm({ name: event.target.value })} />
          </label>
          <div className="toolbar">
            <button className="button" disabled={createProject.isPending}>{t('createProject')}</button>
            <button className="button secondary" type="button" onClick={() => setForm({ name: '' })}>{t('reset')}</button>
          </div>
          {createProject.error ? <p className="error">{createProject.error.message}</p> : null}
        </form>
      </section>
      <section className="card">
        {projects.isLoading ? <p>{t('loading')}</p> : null}
        {projects.error ? <p className="error">{t('error')}: {projects.error.message}</p> : null}
        {projects.data?.length ? (
          <table className="table">
            <thead>
              <tr>
                <th>{t('name')}</th>
                <th>ID</th>
                <th>{t('createdAt')}</th>
              </tr>
            </thead>
            <tbody>
              {projects.data.map((project) => (
                <tr key={project.id}>
                  <td><strong>{project.name}</strong></td>
                  <td><code>{project.id}</code></td>
                  <td>{new Date(project.createdAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : projects.isLoading ? null : <p className="muted">{t('noData')}</p>}
      </section>
    </div>
  );
}
