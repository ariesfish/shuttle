import { useQuery } from '@tanstack/react-query';
import { api, type Task } from './api';
import { useI18n } from './i18n';

export function TasksPage() {
  const { t } = useI18n();
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: api.listTasks, refetchInterval: 5000 });
  const sorted = [...(tasks.data ?? [])].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

  return (
    <section className="card">
      <h1 className="page-title">{t('navTasks')}</h1>
      <p className="page-description">{t('tasksTitle')}</p>
      {tasks.isLoading ? <p>{t('loading')}</p> : null}
      {tasks.error ? <p className="error">{t('error')}: {tasks.error.message}</p> : null}
      {sorted.length ? (
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>{t('taskType')}</th>
              <th>{t('status')}</th>
              <th>{t('clusterId')}</th>
              <th>{t('result')}</th>
              <th>{t('createdAt')}</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((task) => <TaskRow key={task.id} task={task} />)}
          </tbody>
        </table>
      ) : tasks.isLoading ? null : <p className="muted">{t('noData')}</p>}
    </section>
  );
}

function TaskRow({ task }: { task: Task }) {
  const { t } = useI18n();
  return (
    <tr>
      <td><code>{task.id}</code></td>
      <td>{task.type}</td>
      <td><span className="badge">{task.status}</span>{task.error ? <div className="error">{t('errorMessage')}: {task.error}</div> : null}</td>
      <td><code>{task.clusterId}</code></td>
      <td><JsonBlock value={task.result ?? task.payload} /></td>
      <td>{new Date(task.createdAt).toLocaleString()}</td>
    </tr>
  );
}

function JsonBlock({ value }: { value: unknown }) {
  if (!value) {
    return <span className="muted">-</span>;
  }
  return <pre className="code">{JSON.stringify(value, null, 2)}</pre>;
}
