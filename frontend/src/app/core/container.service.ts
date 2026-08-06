import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { ApiResponse } from './compose.service';
import { CacheStore, CacheView } from './cache-store';
import { CacheBus } from './cache-bus';

export interface ContainerRow {
  id: string;
  name: string;
  status: string;
  usingImage: string;
  createTime: string;
  runningTime: string;
  haveUpdate: boolean;
  updateIgnored: boolean;
}

@Injectable({ providedIn: 'root' })
export class ContainerService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<ContainerRow[]>(() => this.list(), '读取容器失败');
  readonly cache: CacheView<ContainerRow[]> = this.store;

  constructor() {
    this.bus.register('containers', this.store);
  }

  ensureLoaded(): void {
    this.store.ensureLoaded();
  }
  refresh(): void {
    this.store.refresh();
  }

  list(): Observable<ApiResponse<ContainerRow[]>> {
    return this.http.get<ApiResponse<ContainerRow[]>>('/api/containers');
  }
  /** 异步检查更新，返回 taskID 供进度跟踪 */
  checkUpdate(): Observable<ApiResponse<{ taskID: string }>> {
    return this.http.post<ApiResponse<{ taskID: string }>>('/api/containers/check-update', {});
  }
  start(id: string) {
    return this.done(
      this.http.post<ApiResponse<unknown>>(
        '/api/container/' + encodeURIComponent(id) + '/start',
        {},
      ),
    );
  }
  stop(id: string) {
    return this.done(
      this.http.post<ApiResponse<unknown>>(
        '/api/container/' + encodeURIComponent(id) + '/stop',
        {},
      ),
    );
  }
  restart(id: string) {
    return this.done(
      this.http.post<ApiResponse<unknown>>(
        '/api/container/' + encodeURIComponent(id) + '/restart',
        {},
      ),
    );
  }
  remove(id: string, force = false) {
    return this.done(
      this.http.delete<ApiResponse<unknown>>(
        '/api/container/' + encodeURIComponent(id) + '?force=' + force,
      ),
    );
  }
  update(id: string, imageNameAndTag = '', containerName = '') {
    return this.http.post<ApiResponse<unknown>>(
      '/api/container/' + encodeURIComponent(id) + '/update',
      { imageNameAndTag, containerName },
    );
  }
  ignoreUpdate(id: string) {
    return this.http
      .post<ApiResponse<unknown>>('/api/container/' + encodeURIComponent(id) + '/update-ignore', {})
      .pipe(
        tap(r => {
          if (r.code === 200) this.store.refresh();
        }),
      );
  }
  restoreUpdate(id: string) {
    return this.http
      .delete<ApiResponse<unknown>>('/api/container/' + encodeURIComponent(id) + '/update-ignore')
      .pipe(
        tap(r => {
          if (r.code === 200) this.store.refresh();
        }),
      );
  }

  /** 操作成功后：刷新容器自身缓存，并让端口/镜像缓存失效（起停会影响端口占用与镜像使用状态） */
  private done(obs: Observable<ApiResponse<unknown>>): Observable<ApiResponse<unknown>> {
    return obs.pipe(
      tap(r => {
        if (r.code === 200) {
          this.store.refresh();
          this.bus.invalidate(['ports', 'images']);
        }
      }),
    );
  }
}
