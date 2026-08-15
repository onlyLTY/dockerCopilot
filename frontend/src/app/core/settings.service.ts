import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, finalize, of, tap, shareReplay } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface UpdateSettings {
  interval: string;
  options: string[];
}

export interface BackupSettings {
  interval: string;
  options: string[];
}

export interface LogSettings {
  level: string;
  options: string[];
}

export interface ProxySettings {
  githubProxy: string;
}
export interface AppSettings {
  updateCheck: UpdateSettings;
  autoBackup: BackupSettings;
  logLevel: LogSettings;
  retention: number;
  /** 拉取镜像超时（秒），0 表示未配置 */
  pullTimeoutSec?: number;
  hubUrls: string[];
  /** 后端内置默认加速源，供「恢复默认」 */
  defaultHubUrls?: string[];
  proxy: ProxySettings;
}

export interface AppSettingsUpdate {
  updateCheckInterval?: string;
  autoBackupInterval?: string;
  logLevel?: string;
  retention?: number;
  pullTimeoutSec?: number;
  hubUrls?: string[];
  proxy?: ProxySettings;
}

export interface RuntimeSettings {
  updateInterval: string;
  backupInterval: string;
  logLevel: string;
  retention: number;
  pullTimeoutSec: number;
}

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly http = inject(HttpClient);
  readonly runtime = signal<RuntimeSettings>({
    updateInterval: '',
    backupInterval: '',
    logLevel: 'info',
    retention: 10,
    pullTimeoutSec: 0,
  });
  private loaded = false;
  private cachedSnapshot: AppSettings | null = null;
  private cachedAt = 0;
  private cacheVersion = 0;
  private settingsRequest$: Observable<ApiResponse<AppSettings>> | null = null;
  private readonly cacheTtlMs = 30_000;

  constructor() {
    this.load();
  }

  getAll(force = false): Observable<ApiResponse<AppSettings>> {
    const now = Date.now();
    if (!force && this.cachedSnapshot && now - this.cachedAt < this.cacheTtlMs) {
      return of({ code: 200, msg: '', data: this.cachedSnapshot });
    }
    if (!this.settingsRequest$) {
      const requestVersion = this.cacheVersion;
      this.settingsRequest$ = this.http.get<ApiResponse<AppSettings>>('/api/settings').pipe(
        tap(result => {
          if (requestVersion !== this.cacheVersion) return;
          if (result.code === 200 && result.data) {
            this.cachedSnapshot = result.data;
            this.cachedAt = Date.now();
            this.applySnapshot(result.data);
          }
        }),
        finalize(() => {
          this.settingsRequest$ = null;
        }),
        shareReplay({ bufferSize: 1, refCount: false }),
      );
    }
    return this.settingsRequest$;
  }

  refresh(): Observable<ApiResponse<AppSettings>> {
    this.invalidate();
    return this.getAll(true);
  }

  invalidate(): void {
    this.cacheVersion++;
    this.cachedSnapshot = null;
    this.cachedAt = 0;
  }

  saveAll(body: AppSettingsUpdate): Observable<ApiResponse<AppSettings>> {
    const requestVersion = ++this.cacheVersion;
    return this.http.put<ApiResponse<AppSettings>>('/api/settings', body).pipe(
      tap(r => {
        if (requestVersion !== this.cacheVersion) return;
        if (r.code === 200 && r.data) {
          this.cachedSnapshot = r.data;
          this.cachedAt = Date.now();
          this.applySnapshot(r.data);
        }
      }),
    );
  }

  load(): void {
    if (this.loaded) return;
    this.loaded = true;
    this.getAll().subscribe({
      next: result => {
        if (result.code === 200 && result.data) this.applySnapshot(result.data);
        else this.loaded = false;
      },
      error: () => {
        this.loaded = false;
      },
    });
  }

  setRuntime(value: Partial<RuntimeSettings>): void {
    this.runtime.update(current => ({ ...current, ...value }));
  }

  private applySnapshot(data: AppSettings): void {
    this.runtime.set({
      updateInterval: data.updateCheck?.interval ?? '',
      backupInterval: data.autoBackup?.interval ?? '',
      logLevel: data.logLevel?.level ?? 'info',
      retention: data.retention ?? 10,
      pullTimeoutSec: data.pullTimeoutSec ?? 0,
    });
  }
}
