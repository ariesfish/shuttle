import { useQuery } from '@tanstack/react-query';
import { api } from './api';
import { useI18n } from './i18n';

export function AuditPage() {
  const { t } = useI18n();
  const records = useQuery({ queryKey: ['audit'], queryFn: api.listAuditRecords, refetchInterval: 10000 });
  const sorted = [...(records.data ?? [])].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

  return (
    <section className="card">
      <h1 className="page-title">{t('auditTitle')}</h1>
      <p className="page-description">{t('auditDescription')}</p>
      {records.isLoading ? <p>{t('loading')}</p> : null}
      {records.error ? <p className="error">{t('error')}: {records.error.message}</p> : null}
      {sorted.length ? (
        <table className="table">
          <thead>
            <tr>
              <th>{t('actor')}</th>
              <th>{t('action')}</th>
              <th>{t('resource')}</th>
              <th>{t('metadata')}</th>
              <th>{t('createdAt')}</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((record) => (
              <tr key={record.id}>
                <td>{record.actor}</td>
                <td><span className="badge muted">{record.action}</span></td>
                <td><code>{record.resource}</code></td>
                <td>{record.metadata ? <pre className="code">{JSON.stringify(record.metadata, null, 2)}</pre> : <span className="muted">-</span>}</td>
                <td>{new Date(record.createdAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : records.isLoading ? null : <p className="muted">{t('noData')}</p>}
    </section>
  );
}
