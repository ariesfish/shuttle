import { useMemo, useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Boxes, ClipboardList, Database, FileClock, FolderKanban, Languages, Rocket } from 'lucide-react';
import { useI18n } from './i18n';
import { ClustersPage } from './ClustersPage';
import { ProjectsPage } from './ProjectsPage';
import { ArtifactsPage } from './ArtifactsPage';
import { ServingAppsPage } from './ServingAppsPage';
import { TasksPage } from './TasksPage';
import { AuditPage } from './AuditPage';
import { getApiSettings, saveApiSettings } from './api';

type Page = 'clusters' | 'projects' | 'artifacts' | 'servingApps' | 'tasks' | 'audit';

const pageKeys: Record<Page, string> = {
  clusters: 'navClusters',
  projects: 'navProjects',
  artifacts: 'navArtifacts',
  servingApps: 'navServingApps',
  tasks: 'navTasks',
  audit: 'navAudit',
};

const pageIcons = {
  clusters: Boxes,
  projects: FolderKanban,
  artifacts: Database,
  servingApps: Rocket,
  tasks: ClipboardList,
  audit: FileClock,
};

export function App() {
  const [queryClient] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}

function AppContent() {
  const { t, locale, setLocale } = useI18n();
  const [page, setPage] = useState<Page>('clusters');
  const [settings, setSettings] = useState(getApiSettings);
  const [saved, setSaved] = useState(false);
  const navItems = useMemo(() => Object.keys(pageKeys) as Page[], []);

  function persistSettings() {
    saveApiSettings(settings.baseUrl, settings.token);
    setSaved(true);
    window.setTimeout(() => setSaved(false), 1200);
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div>
          <h1 className="brand-title">{t('appTitle')}</h1>
          <p className="brand-subtitle">{t('appSubtitle')}</p>
        </div>
        <nav className="nav">
          {navItems.map((item) => {
            const Icon = pageIcons[item];
            return (
              <button key={item} className={`nav-button ${page === item ? 'active' : ''}`} onClick={() => setPage(item)}>
                <Icon size={16} />
                {t(pageKeys[item])}
              </button>
            );
          })}
        </nav>
      </aside>
      <main className="main">
        <div className="topbar">
          <div className="settings">
            <label className="field">
              <span className="label">{t('apiBaseUrl')}</span>
              <input className="input" value={settings.baseUrl} onChange={(event) => setSettings({ ...settings, baseUrl: event.target.value })} />
            </label>
            <label className="field">
              <span className="label">{t('authToken')}</span>
              <input className="input" type="password" value={settings.token} onChange={(event) => setSettings({ ...settings, token: event.target.value })} />
            </label>
            <button className="button secondary" onClick={persistSettings}>{saved ? t('saved') : t('save')}</button>
          </div>
          <div className="toolbar">
            <Languages size={18} />
            <select className="select" value={locale} onChange={(event) => setLocale(event.target.value as 'zh' | 'en')}>
              <option value="zh">{t('chinese')}</option>
              <option value="en">{t('english')}</option>
            </select>
          </div>
        </div>
        {page === 'clusters' ? <ClustersPage /> : null}
        {page === 'projects' ? <ProjectsPage /> : null}
        {page === 'artifacts' ? <ArtifactsPage /> : null}
        {page === 'servingApps' ? <ServingAppsPage /> : null}
        {page === 'tasks' ? <TasksPage /> : null}
        {page === 'audit' ? <AuditPage /> : null}
        {page !== 'clusters' && page !== 'projects' && page !== 'artifacts' && page !== 'servingApps' && page !== 'tasks' && page !== 'audit' ? <Placeholder title={t(pageKeys[page])} /> : null}
      </main>
    </div>
  );
}

function Placeholder({ title }: { title: string }) {
  const { t } = useI18n();
  return (
    <section className="card placeholder">
      <div>
        <h2>{title}</h2>
        <p>{t('comingSoon')}</p>
      </div>
    </section>
  );
}
