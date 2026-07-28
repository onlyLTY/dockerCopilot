import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { ApiResponse } from './compose.service';
import { CacheStore, CacheView } from './cache-store';
import { CacheBus } from './cache-bus';

export interface ImageRow { id: string; name: string; tag: string; size: string; inUsed: boolean; createTime: string; }

export interface ImageCleanupResult { deleted: number; skipped: number; errors: string[]; }

@Injectable({ providedIn: 'root' })
export class ImageService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<ImageRow[]>(() => this.list(), '读取镜像失败');
  readonly cache: CacheView<ImageRow[]> = this.store;

  constructor() { this.bus.register('images', this.store); }

  ensureLoaded(): void { this.store.ensureLoaded(); }
  refresh(): void { this.store.refresh(); }

  list(): Observable<ApiResponse<ImageRow[]>> { return this.http.get<ApiResponse<ImageRow[]>>('/api/images'); }
  remove(id: string, force = false): Observable<ApiResponse<Record<string, unknown>>> {
    return this.done(this.http.delete<ApiResponse<Record<string, unknown>>>(`/api/image/${encodeURIComponent(id)}?force=${force}`));
  }
  cleanup(kind: 'untagged' | 'unused'): Observable<ApiResponse<ImageCleanupResult>> {
    return this.done(this.http.post<ApiResponse<ImageCleanupResult>>('/api/images/prune?kind=' + kind, {}));
  }

  /** 删除/清理成功后：刷新镜像自身缓存，并让容器缓存失效（更新标记可能变化） */
  private done<T>(obs: Observable<ApiResponse<T>>): Observable<ApiResponse<T>> {
    return obs.pipe(tap(r => { if (r.code === 200) { this.store.refresh(); this.bus.invalidate(['containers']); } }));
  }
}
