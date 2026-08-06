import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { ApiResponse } from './compose.service';
import { CacheStore, CacheView } from './cache-store';
import { CacheBus } from './cache-bus';
import { builtInImageIcons } from './built-in-image-icons';

export interface IconMap {
  [name: string]: string;
}

@Injectable({ providedIn: 'root' })
export class IconService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  // 图标变更低频：不按 TTL 自动过期，容器页预加载后端口/镜像页可复用，仅增删时 refresh/invalidate
  private readonly store = new CacheStore<IconMap>(() => this.list(), '读取图标失败', { ttlMs: 0 });
  readonly cache: CacheView<IconMap> = this.store;

  constructor() {
    this.bus.register('icons', this.store);
  }

  ensureLoaded(): void {
    this.store.ensureLoaded();
  }
  refresh(): void {
    this.store.refresh();
  }

  list(): Observable<ApiResponse<IconMap>> {
    return this.http.get<ApiResponse<IconMap>>('/api/icons');
  }
  upload(imageName: string, file: File, containerName = ''): Observable<ApiResponse<string>> {
    const form = new FormData();
    form.append('imageName', imageName);
    form.append('containerName', containerName);
    form.append('file', file);
    return this.done(this.http.post<ApiResponse<string>>('/api/icons', form));
  }
  remove(imageName: string): Observable<ApiResponse<unknown>> {
    return this.done(
      this.http.delete<ApiResponse<unknown>>(
        '/api/icons?imageName=' + encodeURIComponent(imageName),
      ),
    );
  }

  /** 图标增删后：刷新图标自身，并让容器/镜像/端口缓存失效（这些页面卡片图标依赖该 map） */
  private done<T>(obs: Observable<ApiResponse<T>>): Observable<ApiResponse<T>> {
    return obs.pipe(
      tap(r => {
        if (r.code === 200) {
          this.store.refresh();
          this.bus.invalidate(['containers', 'images', 'ports']);
        }
      }),
    );
  }
  /** 无自定义/内置匹配时的默认资源图标 */
  defaultIcon(): string {
    return 'assets/imageIcons/defaultIcon.png';
  }
  url(path: string): string {
    return path || this.defaultIcon();
  }
  actionIcon(name: string): string {
    return 'assets/icons/' + name + '.svg';
  }
  imageIcon(imageName: string): string {
    const file =
      builtInImageIcons[this.normalizeRepository(imageName)] || builtInImageIcons['default'];
    return 'assets/imageIcons/' + file;
  }
  resolve(imageName: string, custom: IconMap): string {
    const repository = this.normalizeRepository(imageName);
    const customKey = Object.keys(custom).find(
      name => this.normalizeRepository(name) === repository,
    );
    return (customKey && custom[customKey]) || this.imageIcon(imageName);
  }
  builtInIcons(): [string, string][] {
    return Object.entries(builtInImageIcons)
      .filter(([name]) => name !== 'default')
      .map(([name, file]) => [name, 'assets/imageIcons/' + file]);
  }
  normalizeRepository(imageName: string): string {
    let value = (imageName || '').trim().toLowerCase().split('@')[0];
    const lastSlash = value.lastIndexOf('/');
    const lastColon = value.lastIndexOf(':');
    if (lastColon > lastSlash) value = value.slice(0, lastColon);
    const parts = value.split('/');
    if (
      parts.length > 1 &&
      (parts[0].includes('.') || parts[0].includes(':') || parts[0] === 'localhost')
    )
      value = parts.slice(1).join('/');
    return value.replace(/^\/+|\/+$/g, '');
  }
}
