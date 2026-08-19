import { HttpEvent, HttpHandlerFn, HttpInterceptorFn, HttpRequest, HttpResponse } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import {
  clone,
  previewBackups,
  previewComposeYaml,
  previewContainers,
  previewImages,
  previewLogs,
  previewPorts,
  previewProjects,
  previewSettings,
  previewVersion,
  projectsData,
} from './preview-fixtures';
import { TaskItem } from '../core/task.service';

interface PreviewResponse<T = unknown> {
  code: number;
  msg: string;
  data: T;
}

interface PreviewProgress {
  taskID: string;
  percentage: number;
  message: string;
  name: string;
  detailMsg: string;
  resourceID?: string;
  isDone: boolean;
  failed?: boolean;
  canceled?: boolean;
  timedOut?: boolean;
  refresh?: boolean;
  startedAt: number;
  endedAt?: number;
  durationMs?: number;
  updatedAt: number;
}

const containers = clone(previewContainers);
const images = clone(previewImages);
const projects = clone(previewProjects);
const backups = [...previewBackups];
const settings = clone(previewSettings);
const progress = new Map<string, PreviewProgress>();
let sequence = 1;

const seededProgress: PreviewProgress[] = [
  {
    taskID: 'preview-task-001', percentage: 72, message: '正在检查镜像层', name: '更新 edge-proxy', detailMsg: '预览任务仍在运行中', resourceID: 'preview-nginx-001', isDone: false, refresh: true, startedAt: Date.now() - 45000, updatedAt: Date.now(),
  },
  {
    taskID: 'preview-task-002', percentage: 100, message: '备份已完成', name: '创建 YAML 容器备份', detailMsg: '已写入本地预览数据', isDone: true, refresh: true, startedAt: Date.now() - 68000, endedAt: Date.now() - 60000, durationMs: 8000, updatedAt: Date.now() - 60000,
  },
];
for (const item of seededProgress) progress.set(item.taskID, item);

function response<T>(data: T, msg = ''): HttpResponse<PreviewResponse<T>> {
  return new HttpResponse({ status: 200, body: { code: 200, msg, data } });
}

function task(title: string, refresh = true, resourceID?: string): { taskID: string } {
  const taskID = `preview-task-${String(++sequence).padStart(3, '0')}`;
  const startedAt = Date.now();
  const endedAt = startedAt;
  progress.set(taskID, {
    taskID, percentage: 100, message: '预览任务已完成', name: title, detailMsg: '本地预览模式不会执行 Docker 操作',
    resourceID, isDone: true, refresh, startedAt, endedAt, durationMs: 0, updatedAt: endedAt,
  });
  return { taskID };
}

function pathOf(req: HttpRequest<unknown>): string {
  return new URL(req.urlWithParams, window.location.origin).pathname.replace(/\/+/g, '/');
}

function bodyOf(req: HttpRequest<unknown>): any {
  return req.body && typeof req.body === 'object' ? req.body : {};
}

function idFrom(path: string, segment: number): string {
  return decodeURIComponent(path.split('/')[segment] || '');
}

function updateContainer(id: string, mutate: (item: typeof containers[number]) => void): boolean {
  const item = containers.find(value => value.id === id);
  if (!item) return false;
  mutate(item);
  return true;
}

function composeFile(projectID: string, filename: string): string {
  if (projectID === 'preview-stack' && filename === 'compose.yaml') return previewComposeYaml;
  if (filename === '.env') return 'APP_ENV=preview\nHTTP_PORT=8080\n';
  return '# Preview compose file\nservices: {}\n';
}

function progressList(): PreviewProgress[] {
  return [...progress.values()].sort((a, b) => b.updatedAt - a.updatedAt);
}

function handle(req: HttpRequest<unknown>): HttpResponse<PreviewResponse<unknown>> {
  const path = pathOf(req);
  const method = req.method.toUpperCase();
  const body = bodyOf(req);

  if (path === '/api/auth') return response({ jwt: 'preview-token' });
  if (path === '/api/version') return response(previewVersion);
  if (path === '/api/icons') return response({});
  if (path === '/api/containers' && method === 'GET') return response(clone(containers));
  if (path === '/api/images' && method === 'GET') return response(clone(images));
  if (path === '/api/ports' && method === 'GET') {
    return response({ ports: clone(previewPorts), conflicts: ['0.0.0.0:8080/tcp'], warnings: ['预览数据故意保留一个端口冲突，方便测试提示状态。'] });
  }
  if (path === '/api/compose/projects' && method === 'GET') {
    const data = projectsData();
    data.projects = clone(projects);
    data.summary = {
      total: projects.length,
      using: projects.filter(item => item.status === 'using').length,
      stopped: projects.filter(item => item.status === 'stopped').length,
      unused: projects.filter(item => item.status === 'unused').length,
      unknown: projects.filter(item => item.status === 'unknown').length,
    };
    return response(data);
  }
  if (path === '/api/container/listBackups' && method === 'GET') return response([...backups]);
  if (path === '/api/container/backup-settings' && method === 'GET') return response({ retention: settings.retention });
  if (path === '/api/settings' && method === 'GET') return response(clone(settings));
  if (path === '/api/settings' && method === 'PUT') {
    const update = body as any;
    if (typeof update['retention'] === 'number') settings.retention = update['retention'];
    if (typeof update['pullTimeoutSec'] === 'number') settings.pullTimeoutSec = update['pullTimeoutSec'];
    if (typeof update['updateCheckInterval'] === 'string') settings.updateCheck.interval = update['updateCheckInterval'];
    if (typeof update['autoBackupInterval'] === 'string') settings.autoBackup.interval = update['autoBackupInterval'];
    if (typeof update['logLevel'] === 'string') settings.logLevel.level = update['logLevel'];
    if (Array.isArray(update['hubUrls'])) settings.hubUrls = update['hubUrls'].map(String);
    if (update['proxy'] && typeof update['proxy'] === 'object') settings.proxy = { githubProxy: String((update['proxy'] as { githubProxy?: unknown }).githubProxy || '') };
    return response(clone(settings));
  }
  if (path === '/api/logs' && method === 'GET') {
    const level = new URL(req.urlWithParams, window.location.origin).searchParams.get('level');
    const entries = previewLogs.filter(item => !level || item.level === level);
    return response({ entries });
  }
  if (path === '/api/progress/list' && method === 'GET') return response(progressList());
  if (path.startsWith('/api/progress/') && method === 'GET') {
    const item = progress.get(idFrom(path, 3));
    return item ? response(item) : response(null, '预览任务不存在');
  }
  if (path.startsWith('/api/progress/') && path.endsWith('/cancel') && method === 'POST') {
    const taskID = idFrom(path, 3);
    const item = progress.get(taskID);
    if (item) {
      item.isDone = true;
      item.canceled = true;
      item.message = '预览任务已取消';
      item.endedAt = Date.now();
      item.durationMs = Math.max(0, item.endedAt - item.startedAt);
      item.updatedAt = item.endedAt;
    }
    return response({});
  }
  if (path.startsWith('/api/progress/') && method === 'DELETE') {
    progress.delete(idFrom(path, 3));
    return response({});
  }
  if (path === '/api/progress/clear' && method === 'DELETE') {
    for (const [key, item] of progress) if (item.isDone) progress.delete(key);
    return response({});
  }

  if (path === '/api/containers/check-update' && method === 'POST') return response(task('检查容器更新'));
  if (path === '/api/containers/remove/batch' && method === 'POST') {
    const ids = Array.isArray(body.containerIds) ? body.containerIds.map(String) : [];
    for (const id of ids) {
      const index = containers.findIndex(item => item.id === id);
      if (index >= 0) containers.splice(index, 1);
    }
    return response(task('批量删除容器'));
  }
  if (path === '/api/images/prune' && method === 'POST') return response(task('清理镜像'));

  if (path.startsWith('/api/container/')) {
    const id = idFrom(path, 3);
    if (path.endsWith('/start') && method === 'POST') {
      updateContainer(id, item => { item.status = 'running'; item.runningTime = 'Up less than a minute'; item.cpuUsage = '0.2%'; item.memoryUsage = '6.0 MiB'; });
      return response({});
    }
    if (path.endsWith('/stop') && method === 'POST') {
      updateContainer(id, item => { item.status = 'exited'; item.runningTime = ''; item.cpuUsage = ''; item.memoryUsage = ''; });
      return response({});
    }
    if (path.endsWith('/restart') && method === 'POST') {
      updateContainer(id, item => { item.status = 'running'; item.runningTime = 'Up less than a minute'; });
      return response({});
    }
    if (path.endsWith('/rename') && method === 'POST') {
      updateContainer(id, item => { item.name = String(body.newName || item.name); });
      return response({});
    }
    if (path.endsWith('/update-ignore') && method === 'POST') {
      updateContainer(id, item => { item.updateIgnored = true; });
      return response({});
    }
    if (path.endsWith('/update-ignore') && method === 'DELETE') {
      updateContainer(id, item => { item.updateIgnored = false; });
      return response({});
    }
    if (path.endsWith('/update') && method === 'POST') {
      const item = containers.find(value => value.id === id);
      if (item && typeof body.imageNameAndTag === 'string' && body.imageNameAndTag) item.usingImage = body.imageNameAndTag;
      if (item && typeof body.containerName === 'string' && body.containerName) item.name = body.containerName;
      return response(task(`更新 ${item?.name || id}`, true, id));
    }
    if (method === 'DELETE') {
      const index = containers.findIndex(item => item.id === id);
      if (index >= 0) containers.splice(index, 1);
      return response({});
    }
    if (path.endsWith('/backup') && method === 'GET') return response(task('创建 JSON 容器备份'));
    if (path.endsWith('/backup2compose') && method === 'GET') return response(task('创建 YAML 容器备份'));
    if (path.endsWith('/backup-settings') && method === 'PUT') {
      const value = Number(new URL(req.urlWithParams, window.location.origin).searchParams.get('retention'));
      if (Number.isFinite(value)) settings.retention = value;
      return response({ retention: settings.retention });
    }
    if (path.endsWith('/backups/restore') && method === 'POST') return response(task('恢复容器备份'));
    if (path === '/api/container/backups' && method === 'DELETE') return response(task('删除容器备份'));
  }

  if (path.startsWith('/api/image/') && method === 'DELETE') {
    const id = idFrom(path, 3);
    const index = images.findIndex(item => item.id === id);
    if (index >= 0) images.splice(index, 1);
    return response({});
  }

  if (path.startsWith('/api/compose/projects/')) {
    const parts = path.split('/');
    const projectID = decodeURIComponent(parts[4] || '');
    const project = projects.find(item => item.id === projectID);
    if (parts.length === 6 && parts[5] === 'files' && method === 'GET') return response(project?.files || []);
    if (parts[5] === 'files' && parts.length >= 7) {
      const filename = decodeURIComponent(parts.slice(6).join('/'));
      if (method === 'GET') return response({ filename, content: composeFile(projectID, filename), version: project?.files.find(item => item.name === filename)?.version || 'preview-v1' });
      if (method === 'PUT') return response({ version: `preview-v${Date.now()}` });
    }
    if (parts[5] === 'deploy' && parts[6] === 'preview' && method === 'POST') return response({ projectId: projectID, confirmToken: 'preview-confirm-token', risks: [] });
    if (parts[5] === 'deploy' && method === 'POST') return response(task(`部署 ${project?.name || projectID}`));
    if (parts[5] === 'backup' && method === 'POST') return response({ filename: `${project?.name || projectID}-backup.yaml` });
    if (parts[5] === 'cleanup' && parts[6] === 'preview' && method === 'POST') return response({ previewToken: 'preview-cleanup-token', items: [] });
    if (parts[5] === 'cleanup' && method === 'POST') return response({});
  }
  if (path === '/api/compose/projects/backup' && method === 'POST') return response(task('批量备份 Compose 项目'));
  if (path === '/api/compose/projects/cleanup/batch' && method === 'POST') return response(task('批量删除 Compose 项目'));
  if (path === '/api/compose/projects' && method === 'POST') {
    const id = `preview-project-${Date.now()}`;
    projects.push({ id, name: String(body.projectName || 'preview-project'), root: `/compose/${String(body.projectName || 'preview-project')}`, files: [{ name: String(body.filename || 'compose.yaml'), size: String(body.content || '').length, modifiedAt: '刚刚', valid: true, version: 'preview-v1' }], status: 'unused', containers: [], ports: [], warnings: [] });
    return response({ projectId: id, version: 'preview-v1' });
  }
  if (path === '/api/compose/validate' && method === 'POST') return response({ valid: true, name: 'preview-project', services: ['web', 'cache'] });

  return response({}, '预览模式未实现此接口');
}

export function consumeSessionExpiredFlag(): boolean {
  return false;
}

export const authInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  if (!req.url.startsWith('/api/')) return next(req);
  return of(handle(req));
};

export function previewTaskSnapshot(): TaskItem[] {
  return progressList().map(item => ({
    taskID: item.taskID, title: item.name, percentage: item.percentage, message: item.message, detailMsg: item.detailMsg,
    isDone: item.isDone, failed: !!item.failed, canceled: !!item.canceled, timedOut: !!item.timedOut, refresh: !!item.refresh,
    createdAt: item.startedAt, updatedAt: item.updatedAt, startedAt: item.startedAt, endedAt: item.endedAt, durationMs: item.durationMs, resourceID: item.resourceID, steps: [],
  }));
}
