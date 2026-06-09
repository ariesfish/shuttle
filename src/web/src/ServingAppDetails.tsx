import { type ObservabilitySummary, type ServingApplicationTransition, type Task } from './api';
import { useI18n } from './i18n';

export function ServingAppDetails({ selectedAppId, summary, summaryError, transitions, diagnosticsTask }: { selectedAppId: string; summary?: ObservabilitySummary; summaryError?: string; transitions?: ServingApplicationTransition[]; diagnosticsTask?: Task }) {
  const { t } = useI18n();
  return (
    <section className="card">
      <h2>History & Diagnostics</h2>
      {!selectedAppId ? <p className="muted">Select a Serving Application to inspect transitions and diagnostics.</p> : null}
      {selectedAppId ? (
        <div className="grid">
          <ObservabilitySummaryCard summary={summary} error={summaryError} />
          <div>
            <h3>Transitions</h3>
            {transitions?.slice(-8).reverse().map((transition) => (
              <div key={transition.id} className="card" style={{ marginBottom: 8 }}>
                <strong>{transition.from || '-'} → {transition.to}</strong> <span className="badge muted">{transition.actor}</span>
                <div className="muted"><code>{transition.taskId || transition.id}</code></div>
                {transition.reason ? <div>{transition.reason}</div> : null}
              </div>
            )) ?? <p className="muted">{t('noData')}</p>}
          </div>
          <DiagnosticsTask task={diagnosticsTask} />
        </div>
      ) : null}
    </section>
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
