import { useMemo } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, type EndpointRegistryEntry, type ServingApplication, type ServingRecipe, type Task } from './api';

export type ServingApplicationAction = 'preview' | 'apply' | 'redeploy' | 'retire' | 'diagnostics';

export interface ServingApplicationControlOptions {
  apps?: ServingApplication[];
  recipes?: ServingRecipe[];
  tasks?: Task[];
  endpoints?: EndpointRegistryEntry[];
  selectedAppId: string;
  confirmExperimental?: (message: string) => boolean;
}

export function useServingApplicationControl(options: ServingApplicationControlOptions) {
  const queryClient = useQueryClient();
  const confirmExperimental = options.confirmExperimental ?? ((message: string) => confirm(message));

  const endpointsByApp = useMemo(() => endpointMap(options.endpoints ?? []), [options.endpoints]);
  const latestDiagnosticsTask = useMemo(() => latestDiagnosticsForApp(options.tasks ?? [], options.selectedAppId), [options.selectedAppId, options.tasks]);

  const actionMutation = useMutation({
    mutationFn: ({ appId, action }: { appId: string; action: ServingApplicationAction }) => executeServingApplicationAction({ appId, action, apps: options.apps ?? [], recipes: options.recipes ?? [], confirmExperimental }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['apps'] });
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });

  return {
    endpointsByApp,
    latestDiagnosticsTask,
    actionMutation,
  };
}

export async function executeServingApplicationAction(input: { appId: string; action: ServingApplicationAction; apps: ServingApplication[]; recipes: ServingRecipe[]; confirmExperimental: (message: string) => boolean }) {
  const app = input.apps.find((candidate) => candidate.id === input.appId);
  const recipe = recipeForApp(input.recipes, app);
  if (requiresExperimentalConfirmation(input.action, recipe) && !input.confirmExperimental(experimentalRecipeMessage(recipe))) {
    throw new Error('action cancelled');
  }
  switch (input.action) {
    case 'preview':
      return api.createPreviewTask(input.appId);
    case 'apply':
      return api.createApplyTask(input.appId);
    case 'redeploy':
      return api.createRedeployTask(input.appId);
    case 'diagnostics':
      return api.createDiagnosticsTask(input.appId);
    case 'retire':
      return api.createRetireTask(input.appId);
  }
}

export function endpointMap(endpoints: EndpointRegistryEntry[]) {
  const map = new Map<string, string>();
  for (const endpoint of endpoints) {
    map.set(endpoint.servingApplicationId, endpoint.url);
  }
  return map;
}

export function latestDiagnosticsForApp(tasks: Task[], appId: string) {
  return [...tasks].reverse().find((task) => task.type === 'FetchDiagnostics' && task.payload?.servingApplicationId === appId);
}

function requiresExperimentalConfirmation(action: ServingApplicationAction, recipe?: ServingRecipe) {
  return (action === 'apply' || action === 'redeploy') && recipe?.spec.support.status === 'experimental';
}

function experimentalRecipeMessage(recipe?: ServingRecipe) {
  return `Recipe is experimental: ${recipe?.spec.support.warning || recipe?.metadata.name || 'unknown recipe'}`;
}

function recipeForApp(recipes: ServingRecipe[], app?: ServingApplication) {
  if (!app) return undefined;
  return recipes.find((recipe) => recipe.metadata.id === app.runtime.recipe);
}
