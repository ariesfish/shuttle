import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, type CreateServingApplicationInput } from './api';
import { ServingAppCreateForm } from './ServingAppCreateForm';
import { ServingAppDetails } from './ServingAppDetails';
import { RecentTasks, ServingAppsTable } from './ServingAppRows';
import { useI18n } from './i18n';
import { useServingApplicationControl } from './servingApplicationControl';

export function ServingAppsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.listProjects });
  const clusters = useQuery({ queryKey: ['clusters'], queryFn: api.listClusters });
  const artifacts = useQuery({ queryKey: ['artifacts'], queryFn: api.listModelArtifacts });
  const recipes = useQuery({ queryKey: ['recipes'], queryFn: api.listRecipes });
  const apps = useQuery({ queryKey: ['apps'], queryFn: api.listServingApplications, refetchInterval: 2000 });
  const tasks = useQuery({ queryKey: ['tasks'], queryFn: api.listTasks, refetchInterval: 2000 });
  const endpoints = useQuery({ queryKey: ['endpoints'], queryFn: api.listEndpoints, refetchInterval: 2000 });
  const poolSummaries = useQuery({ queryKey: ['acceleratorPoolSummaries'], queryFn: () => api.listAcceleratorPoolSummaries(), refetchInterval: 5000 });

  const [selectedArtifactId, setSelectedArtifactId] = useState('');
  const [selectedAppId, setSelectedAppId] = useState('');

  const creationPlans = useQuery({
    queryKey: ['app-plans', selectedArtifactId],
    queryFn: () => api.listServingApplicationCreationPlans(selectedArtifactId),
    enabled: Boolean(selectedArtifactId),
  });

  const createApp = useMutation({
    mutationFn: (input: CreateServingApplicationInput) => api.createServingApplication(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] });
    },
  });

  const control = useServingApplicationControl({ apps: apps.data, recipes: recipes.data, tasks: tasks.data, endpoints: endpoints.data, selectedAppId });
  const transitions = useQuery({
    queryKey: ['app-transitions', selectedAppId],
    queryFn: () => api.listServingApplicationTransitions(selectedAppId),
    enabled: Boolean(selectedAppId),
    refetchInterval: 2000,
  });
  const tuningRecords = useQuery({
    queryKey: ['tuning-records', selectedAppId],
    queryFn: () => api.listTuningRecords(selectedAppId),
    enabled: Boolean(selectedAppId),
    refetchInterval: 5000,
  });
  const createTuningRecord = useMutation({
    mutationFn: api.createTuningRecord,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tuning-records', selectedAppId] }),
  });
  const observabilitySummary = useQuery({
    queryKey: ['observability-summary', selectedAppId],
    queryFn: () => api.getObservabilitySummary(selectedAppId),
    enabled: Boolean(selectedAppId),
    refetchInterval: 10000,
  });

  return (
    <div className="grid two">
      <section className="card">
        <h1 className="page-title">{t('servingAppsTitle')}</h1>
        <p className="page-description">{t('servingAppsDescription')}</p>
        <ServingAppCreateForm
          projects={projects.data ?? []}
          clusters={clusters.data ?? []}
          artifacts={artifacts.data ?? []}
          creationPlans={creationPlans.data}
          poolSummaries={poolSummaries.data ?? []}
          creating={createApp.isPending}
          createError={createApp.error?.message}
          onArtifactChange={setSelectedArtifactId}
          onCreate={(input) => createApp.mutate(input)}
        />
      </section>
      <section className="grid">
        <section className="card">
          {apps.isLoading ? <p>{t('loading')}</p> : null}
          {apps.error ? <p className="error">{t('error')}: {apps.error.message}</p> : null}
          {apps.data?.length ? (
            <ServingAppsTable
              apps={apps.data}
              endpointsByApp={control.endpointsByApp}
              selectedAppId={selectedAppId}
              actionPending={control.actionMutation.isPending}
              onSelect={setSelectedAppId}
              onAction={(appId, action) => control.actionMutation.mutate({ appId, action })}
            />
          ) : apps.isLoading ? null : <p className="muted">{t('noData')}</p>}
        </section>
        <ServingAppDetails selectedAppId={selectedAppId} tuningRecords={tuningRecords.data} tuningError={tuningRecords.error?.message || createTuningRecord.error?.message} tuningCreating={createTuningRecord.isPending} onCreateTuningRecord={(reason) => createTuningRecord.mutate({ servingApplicationId: selectedAppId, reason, benchmarkSummary: { source: 'manual-summary' }, plannerSettings: {}, recommendations: [] })} summary={observabilitySummary.data} summaryError={observabilitySummary.error?.message} transitions={transitions.data} diagnosticsTask={control.latestDiagnosticsTask} />
        <RecentTasks tasks={tasks.data} />
      </section>
    </div>
  );
}
