import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { CacheStore, CacheView } from './cache-store';
import { CacheBus } from './cache-bus';

export interface ApiResponse<T> { code: number; msg: string; data: T; }
export interface ComposePort { hostIP: string; hostPort: string; containerPort: string; protocol: string; published: boolean; }
export interface ComposeContainer { id: string; name: string; service: string; state: string; ports: ComposePort[]; }
export interface ComposeFile { name: string; size: number; modifiedAt: string; valid: boolean; version?: string; }
export interface ComposeProject { id: string; name: string; root: string; files: ComposeFile[]; status: 'using' | 'stopped' | 'unused' | 'unknown'; containers: ComposeContainer[]; ports: ComposePort[]; warnings?: string[]; }
export interface ComposeSummary { total: number; using: number; stopped: number; unused: number; unknown: number; }
export interface ProjectsData { summary: ComposeSummary; projects: ComposeProject[]; }
export interface PortUsage { project: string; containerID: string; containerName: string; state: string; hostIP: string; hostPort: string; containerPort: string; protocol: string; published: boolean; conflictKey?: string; }

@Injectable({ providedIn: 'root' })
export class ComposeService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<ProjectsData>(() => this.projects(), '读取项目失败');
  readonly cache: CacheView<ProjectsData> = this.store;

  constructor() { this.bus.register('compose', this.store); }

  ensureLoaded(): void { this.store.ensureLoaded(); }
  refresh(): void { this.store.refresh(); }

  projects(): Observable<ApiResponse<ProjectsData>> { return this.http.get<ApiResponse<ProjectsData>>('/api/compose/projects'); }
  createProject(projectName: string, filename: string, content: string): Observable<ApiResponse<{ projectId: string; version: string }>> {
    return this.http.post<ApiResponse<{ projectId: string; version: string }>>('/api/compose/projects', { projectName, filename, content })
      .pipe(tap(r => { if (r.code === 200) { this.store.refresh(); this.bus.invalidate(['containers', 'ports', 'images']); } }));
  }
  ports(): Observable<ApiResponse<{ ports: PortUsage[]; conflicts: string[]; warnings: string[] }>> { return this.http.get<ApiResponse<{ ports: PortUsage[]; conflicts: string[]; warnings: string[] }>>('/api/ports'); }
  files(projectId: string): Observable<ApiResponse<ComposeFile[]>> { return this.http.get<ApiResponse<ComposeFile[]>>(`/api/compose/projects/${projectId}/files`); }
  file(projectId: string, filename: string): Observable<ApiResponse<{ filename: string; content: string; version: string }>> {
    return this.http.get<ApiResponse<{ filename: string; content: string; version: string }>>(`/api/compose/projects/${projectId}/files/${encodeURIComponent(filename)}`);
  }
  update(projectId: string, filename: string, content: string, version: string): Observable<ApiResponse<{ version: string }>> {
    return this.http.put<ApiResponse<{ version: string }>>(`/api/compose/projects/${projectId}/files/${encodeURIComponent(filename)}`, { content, version });
  }
  validate(projectId: string, filename: string, content: string): Observable<ApiResponse<{ valid: boolean; name: string; services: string[] }>> {
    return this.http.post<ApiResponse<{ valid: boolean; name: string; services: string[] }>>('/api/compose/validate', { projectId, filename, content });
  }
  deployPreview(projectId: string, filename: string): Observable<ApiResponse<Record<string, unknown>>> {
    return this.http.post<ApiResponse<Record<string, unknown>>>('/api/compose/projects/' + projectId + '/deploy/preview', { projectId, filename });
  }
  // 部署是异步任务：提交仅返回 taskID，真正完成后由 TaskService 轮询到 isDone 时联动刷新缓存
  deploy(projectId: string, filename: string, confirmToken: string, confirmWarnings: boolean): Observable<ApiResponse<Record<string, unknown>>> {
    return this.http.post<ApiResponse<Record<string, unknown>>>('/api/compose/projects/' + projectId + '/deploy', { projectId, filename, confirmToken, confirmWarnings });
  }
}
