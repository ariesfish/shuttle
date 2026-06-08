import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';

type Locale = 'zh' | 'en';
type Dictionary = Record<string, string>;

const dictionaries: Record<Locale, Dictionary> = {
  zh: {
    appTitle: '推理平台',
    appSubtitle: 'Management Plane 控制台',
    navClusters: '集群',
    navProjects: '项目',
    navArtifacts: '模型制品',
    navServingApps: '服务应用',
    navTasks: '任务',
    navAudit: '审计',
    language: '语言',
    chinese: '中文',
    english: 'English',
    clustersTitle: '推理集群',
    clustersDescription: '注册推理集群，查看 Cluster Agent 心跳与能力信息。',
    createCluster: '创建集群',
    name: '名称',
    description: '描述',
    prometheusUrl: 'Prometheus URL',
    grafanaUrl: 'Grafana URL',
    create: '创建',
    reset: '重置',
    clusterId: '集群 ID',
    agentStatus: 'Agent 状态',
    capabilities: '能力',
    noData: '暂无数据',
    loading: '加载中…',
    error: '请求失败',
    installCommand: 'Agent 启动命令',
    copyHint: '复制后在目标推理集群中运行。',
    apiSettings: 'API 设置',
    apiBaseUrl: 'API 地址',
    authToken: 'Auth Token',
    save: '保存',
    saved: '已保存',
    unknown: '未知',
    online: '在线',
    never: '从未',
    lastHeartbeat: '最后心跳',
    version: '版本',
    comingSoon: '即将实现',
    projectsTitle: '项目',
    projectsDescription: '创建 Project，作为用户、权限和 Serving Application 的产品边界。',
    createProject: '创建项目',
    artifactsTitle: '模型制品',
    artifactsDescription: '登记已缓存的 Model Artifact，Phase 1 不负责模型下载和分发。',
    createArtifact: '登记模型制品',
    family: '模型族',
    variant: '变体',
    revision: 'Revision',
    pvcName: 'PVC 名称',
    pvcMountPath: 'PVC 挂载路径',
    pvcModelPath: 'PVC 模型路径',
    hostCachePath: '节点缓存路径',
    quantization: '量化',
    createdAt: '创建时间',
  },
  en: {
    appTitle: 'Inference Platform',
    appSubtitle: 'Management Plane Console',
    navClusters: 'Clusters',
    navProjects: 'Projects',
    navArtifacts: 'Model Artifacts',
    navServingApps: 'Serving Apps',
    navTasks: 'Tasks',
    navAudit: 'Audit',
    language: 'Language',
    chinese: '中文',
    english: 'English',
    clustersTitle: 'Inference Clusters',
    clustersDescription: 'Register inference clusters and inspect Cluster Agent heartbeat and capabilities.',
    createCluster: 'Create Cluster',
    name: 'Name',
    description: 'Description',
    prometheusUrl: 'Prometheus URL',
    grafanaUrl: 'Grafana URL',
    create: 'Create',
    reset: 'Reset',
    clusterId: 'Cluster ID',
    agentStatus: 'Agent Status',
    capabilities: 'Capabilities',
    noData: 'No data',
    loading: 'Loading…',
    error: 'Request failed',
    installCommand: 'Agent Command',
    copyHint: 'Copy and run this inside the target inference cluster.',
    apiSettings: 'API Settings',
    apiBaseUrl: 'API Base URL',
    authToken: 'Auth Token',
    save: 'Save',
    saved: 'Saved',
    unknown: 'Unknown',
    online: 'Online',
    never: 'Never',
    lastHeartbeat: 'Last Heartbeat',
    version: 'Version',
    comingSoon: 'Coming soon',
    projectsTitle: 'Projects',
    projectsDescription: 'Create Projects as the product boundary for users, permissions, and Serving Applications.',
    createProject: 'Create Project',
    artifactsTitle: 'Model Artifacts',
    artifactsDescription: 'Register cached Model Artifacts. Phase 1 does not download or distribute model weights.',
    createArtifact: 'Register Model Artifact',
    family: 'Family',
    variant: 'Variant',
    revision: 'Revision',
    pvcName: 'PVC Name',
    pvcMountPath: 'PVC Mount Path',
    pvcModelPath: 'PVC Model Path',
    hostCachePath: 'Host Cache Path',
    quantization: 'Quantization',
    createdAt: 'Created At',
  },
};

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(() => (localStorage.getItem('locale') as Locale) || 'zh');
  const value = useMemo(() => ({
    locale,
    setLocale: (next: Locale) => {
      localStorage.setItem('locale', next);
      setLocale(next);
    },
    t: (key: string) => dictionaries[locale][key] ?? key,
  }), [locale]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const value = useContext(I18nContext);
  if (!value) {
    throw new Error('useI18n must be used within I18nProvider');
  }
  return value;
}
