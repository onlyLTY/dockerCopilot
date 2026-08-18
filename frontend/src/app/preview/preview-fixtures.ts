import { ContainerRow } from '../core/container.service';
import { ImageRow } from '../core/image.service';
import { ComposeProject, PortUsage, ProjectsData } from '../core/compose.service';
import { AppSettings } from '../core/settings.service';

export const previewVersion = {
  version: 'v0.9.0-preview',
  buildDate: '2026-08-18 10:00:00',
};

export const previewContainers: ContainerRow[] = [
  {
    id: 'preview-nginx-001', name: 'edge-proxy', status: 'running', usingImage: 'nginx:1.27-alpine', createImage: 'nginx:1.27-alpine',
    createTime: '2026-08-18 08:12:00', runningTime: 'Up 2 hours', cpuUsage: '1.8%', memoryUsage: '12.4 MiB', haveUpdate: true, updateIgnored: false,
  },
  {
    id: 'preview-redis-002', name: 'cache', status: 'running', usingImage: 'redis:7-alpine', createImage: 'redis:7-alpine',
    createTime: '2026-08-17 22:31:00', runningTime: 'Up 12 hours', cpuUsage: '0.4%', memoryUsage: '8.1 MiB', haveUpdate: false, updateIgnored: false,
  },
  {
    id: 'preview-postgres-003', name: 'database', status: 'exited', usingImage: 'postgres:16-alpine', createImage: 'postgres:16-alpine',
    createTime: '2026-08-16 14:08:00', runningTime: '', cpuUsage: '', memoryUsage: '', haveUpdate: false, updateIgnored: false,
  },
  {
    id: 'preview-whoami-004', name: 'whoami', status: 'running', usingImage: 'traefik/whoami:v1.10', createImage: 'traefik/whoami:v1.10',
    createTime: '2026-08-15 09:20:00', runningTime: 'Up 1 day', cpuUsage: '0.1%', memoryUsage: '5.6 MiB', haveUpdate: false, updateIgnored: true,
  },
  {
    id: 'preview-watchtower-005', name: 'watchtower', status: 'restarting', usingImage: 'containrrr/watchtower:latest', createImage: 'containrrr/watchtower:latest',
    createTime: '2026-08-12 17:40:00', runningTime: 'Up less than a second', cpuUsage: '0.3%', memoryUsage: '7.2 MiB', haveUpdate: true, updateIgnored: false,
  },
];

export const previewImages: ImageRow[] = [
  { id: 'sha256:nginx-preview', name: 'nginx', tag: '1.27-alpine', repoTags: ['nginx:1.27-alpine'], size: '48.2 MB', inUsed: true, createTime: '2026-08-18' },
  { id: 'sha256:redis-preview', name: 'redis', tag: '7-alpine', repoTags: ['redis:7-alpine'], size: '43.8 MB', inUsed: true, createTime: '2026-08-17' },
  { id: 'sha256:postgres-preview', name: 'postgres', tag: '16-alpine', repoTags: ['postgres:16-alpine'], size: '92.6 MB', inUsed: true, createTime: '2026-08-16' },
  { id: 'sha256:whoami-preview', name: 'traefik/whoami', tag: 'v1.10', repoTags: ['traefik/whoami:v1.10'], size: '6.1 MB', inUsed: true, createTime: '2026-08-15' },
  { id: 'sha256:old-preview', name: 'demo/old-service', tag: '0.8.0', repoTags: ['demo/old-service:0.8.0'], size: '122.4 MB', inUsed: false, createTime: '2026-07-28' },
  { id: 'sha256:dangling-preview', name: '<none>', tag: '<none>', repoTags: [], size: '18.3 MB', inUsed: false, createTime: '2026-07-20' },
];

export const previewComposeYaml = `services:\n  web:\n    image: nginx:1.27-alpine\n    ports:\n      - "8080:80"\n  cache:\n    image: redis:7-alpine\n`;

export const previewProjects: ComposeProject[] = [
  {
    id: 'preview-stack', name: 'demo-stack', image: 'nginx:1.27-alpine', root: '/compose/demo-stack',
    files: [
      { name: 'compose.yaml', size: previewComposeYaml.length, modifiedAt: '2026-08-18 09:12:00', valid: true, version: 'preview-v1' },
      { name: '.env', size: 84, modifiedAt: '2026-08-18 09:10:00', valid: true, version: 'preview-v2' },
    ],
    status: 'using',
    containers: [
      { id: 'preview-nginx-001', name: 'edge-proxy', service: 'web', state: 'running', ports: [{ hostIP: '0.0.0.0', hostPort: '8080', containerPort: '80', protocol: 'tcp', published: true }] },
      { id: 'preview-redis-002', name: 'cache', service: 'cache', state: 'running', ports: [] },
    ],
    ports: [{ hostIP: '0.0.0.0', hostPort: '8080', containerPort: '80', protocol: 'tcp', published: true }], warnings: [],
  },
  {
    id: 'preview-stopped', name: 'stopped-demo', image: 'postgres:16-alpine', root: '/compose/stopped-demo',
    files: [{ name: 'compose.yaml', size: 412, modifiedAt: '2026-08-10 15:44:00', valid: true, version: 'preview-v3' }],
    status: 'stopped',
    containers: [{ id: 'preview-postgres-003', name: 'database', service: 'db', state: 'exited', ports: [] }], ports: [], warnings: ['项目中的数据库容器当前已停止'],
  },
  {
    id: 'preview-unused', name: 'unused-demo', image: 'demo/old-service:0.8.0', root: '/compose/unused-demo',
    files: [{ name: 'compose.yaml', size: 198, modifiedAt: '2026-07-28 11:20:00', valid: true, version: 'preview-v4' }],
    status: 'unused', containers: [], ports: [], warnings: [],
  },
];

export const previewPorts: PortUsage[] = [
  { project: 'demo-stack', containerID: 'preview-nginx-001', containerName: 'edge-proxy', image: 'nginx:1.27-alpine', state: 'running', hostIP: '0.0.0.0', hostPort: '8080', containerPort: '80', protocol: 'tcp', published: true },
  { project: 'demo-stack', containerID: 'preview-redis-002', containerName: 'cache', image: 'redis:7-alpine', state: 'running', hostIP: '127.0.0.1', hostPort: '6379', containerPort: '6379', protocol: 'tcp', published: true },
  { project: '', containerID: 'preview-whoami-004', containerName: 'whoami', image: 'traefik/whoami:v1.10', state: 'running', hostIP: '0.0.0.0', hostPort: '8080', containerPort: '80', protocol: 'tcp', published: true, conflictKey: '0.0.0.0:8080/tcp' },
];

export const previewSettings: AppSettings = {
  updateCheck: { interval: '6h', options: ['off', '30m', '1h', '6h', '12h', '24h'] },
  autoBackup: { interval: '24h', options: ['off', '6h', '12h', '24h', 'week', 'month'] },
  logLevel: { level: 'info', options: ['debug', 'info', 'warn', 'error'] },
  retention: 10, pullTimeoutSec: 300, hubUrls: ['docker.m.daocloud.io', 'docker.1ms.run'], defaultHubUrls: ['docker.m.daocloud.io', 'docker.1ms.run'], proxy: { githubProxy: '' },
};

export const previewBackups = ['container-backup-2026-08-18.json', 'container-backup-2026-08-17.yaml', 'container-backup-2026-08-10.json'];

export const previewLogs = [
  { timestamp: '2026-08-18 09:42:18', level: 'error', message: '预览日志：示例容器 edge-proxy 的健康检查曾短暂超时。' },
  { timestamp: '2026-08-18 09:40:02', level: 'warn', message: '预览日志：发现端口 8080 存在冲突，请前往端口页面查看。' },
  { timestamp: '2026-08-18 09:35:46', level: 'info', message: '预览模式已加载本地 fixture，不会连接 Docker Engine。' },
];

export function clone<T>(value: T): T {
  return structuredClone(value);
}

export function projectsData(): ProjectsData {
  const projects = clone(previewProjects);
  return {
    summary: {
      total: projects.length,
      using: projects.filter(item => item.status === 'using').length,
      stopped: projects.filter(item => item.status === 'stopped').length,
      unused: projects.filter(item => item.status === 'unused').length,
      unknown: projects.filter(item => item.status === 'unknown').length,
    },
    projects,
  };
}
