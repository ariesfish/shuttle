import { useState } from 'react';
import { type ObservabilitySummary, type ProductionObservabilityEntryPoints, type ServingApplicationTransition, type Task, type TuningRecord } from './api';
import { useI18n } from './i18n';

export function ServingAppDetails({ selectedAppId, summary, summaryError, entryPoints, transitions, diagnosticsTask, tuningRecords, tuningError, tuningCreating, onCreateTuningRecord }: { selectedAppId: string; summary?: ObservabilitySummary; summaryError?: string; entryPoints?: ProductionObservabilityEntryPoints; transitions?: ServingApplicationTransition[]; diagnosticsTask?: Task; tuningRecords?: TuningRecord[]; tuningError?: string; tuningCreating: boolean; onCreateTuningRecord: (reason: string) => void }) {
  const { t } = useI18n();
  return (
    <section className="card">
      <h2>History & Diagnostics</h2>
      {!selectedAppId ? <p className="muted">Select a Serving Application to inspect transitions, tuning, and diagnostics.</p> : null}
      {selectedAppId ? (
        <div className="grid">
          <TuningRecords records={tuningRecords} error={tuningError} creating={tuningCreating} onCreate={onCreateTuningRecord} />
          <ObservabilityEntryPointsCard entryPoints={entryPoints} />
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

function TuningRecords({ records, error, creating, onCreate }: { records?: TuningRecord[]; error?: string; creating: boolean; onCreate: (reason: string) => void }) {
  const [reason, setReason] = useState('manual baseline');
  return (
    <div>
      <h3>Tuning Records</h3>
      <div className="toolbar">
        <input className="input" value={reason} onChange={(event) => setReason(event.target.value)} />
        <button className="button secondary" disabled={creating} onClick={() => onCreate(reason)}>Create</button>
      </div>
      {error ? <p className="error">{error}</p> : null}
      {records?.slice(-5).reverse().map((record) => (
        <div key={record.id} className="card" style={{ marginBottom: 8 }}>
          <strong>{record.reason || record.id}</strong> <span className="badge muted">{record.actor}</span>
          <div className="muted">Inventory: <code>{record.acceleratorInventoryRevision}</code></div>
          <div className="muted">Recipe: {record.servingRecipeId}</div>
          {record.recommendations?.length ? <div>{record.recommendations.join(', ')}</div> : null}
        </div>
      )) ?? <p className="muted">No tuning records.</p>}
    </div>
  );
}

function ObservabilityEntryPointsCard({ entryPoints }: { entryPoints?: ProductionObservabilityEntryPoints }) {
  return (
    <div>
      <h3>Observability Entry Points</h3>
      {!entryPoints ? <p className="muted">Loading observability entry points...</p> : null}
      {entryPoints?.links.map((link) => (
        <div key={`${link.type}-${link.name}`} className="card" style={{ marginBottom: 8 }}>
          <strong>{link.name}</strong> <span className="badge muted">{link.type}</span>
          {link.url ? <div><code>{link.url}</code></div> : <div className="muted">No URL configured.</div>}
        </div>
      ))}
      {entryPoints?.alerts?.map((alert) => <p key={alert.reason} className="error">{alert.reason}: {alert.message}</p>)}
      {entryPoints?.telemetryCoverage?.length ? <p className="muted">Coverage: {entryPoints.telemetryCoverage.join(', ')}</p> : null}
    </div>
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
