import { type ServingApplication, type Task } from './api';
import { useI18n } from './i18n';
import { type ServingApplicationAction } from './servingApplicationControl';

export function ServingAppsTable({ apps, endpointsByApp, selectedAppId, actionPending, onSelect, onAction }: { apps: ServingApplication[]; endpointsByApp: Map<string, string>; selectedAppId: string; actionPending: boolean; onSelect: (appId: string) => void; onAction: (appId: string, action: ServingApplicationAction) => void }) {
  const { t } = useI18n();
  return (
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
        {apps.map((app) => (
          <ServingAppRow key={app.id} app={app} endpoint={endpointsByApp.get(app.id)} selected={selectedAppId === app.id} onSelect={() => onSelect(app.id)} onAction={(action) => onAction(app.id, action)} disabled={actionPending} />
        ))}
      </tbody>
    </table>
  );
}

export function RecentTasks({ tasks }: { tasks?: Task[] }) {
  const { t } = useI18n();
  return (
    <section className="card">
      <h2>{t('tasksTitle')}</h2>
      {tasks?.slice(-8).reverse().map((task) => (
        <div key={task.id} className="card" style={{ marginBottom: 8 }}>
          <strong>{task.type}</strong> <span className="badge muted">{task.status}</span>
          <div className="muted"><code>{task.id}</code></div>
          {task.error ? <div className="error">{task.error}</div> : null}
        </div>
      )) ?? <p className="muted">{t('noData')}</p>}
    </section>
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
