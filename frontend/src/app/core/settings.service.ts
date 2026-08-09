import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, map } from 'rxjs';
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
  HTTP_PROXY: string;
  HTTPS_PROXY: string;
  NO_PROXY: string;
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

  constructor() {
    this.load();
  }

  getAll(): Observable<ApiResponse<AppSettings>> {
    return this.http.get<ApiResponse<AppSettings>>('/api/settings');
  }

  saveAll(body: AppSettingsUpdate): Observable<ApiResponse<AppSettings>> {
    return this.http.put<ApiResponse<AppSettings>>('/api/settings', body).pipe(
      tap(r => {
        if (r.code === 200 && r.data) {
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

  // 兼容旧分项调用：设置页等仍可单独读写某一类配置。

  getUpdateSettings(): Observable<ApiResponse<UpdateSettings>> {
    return this.getAll().pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.updateCheck ?? { interval: '', options: [] },
      })),
    );
  }

  setUpdateInterval(interval: string): Observable<ApiResponse<UpdateSettings>> {
    return this.saveAll({ updateCheckInterval: interval }).pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.updateCheck ?? { interval, options: [] },
      })),
    );
  }

  getBackupSettings(): Observable<ApiResponse<BackupSettings>> {
    return this.getAll().pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.autoBackup ?? { interval: '', options: [] },
      })),
    );
  }

  setBackupInterval(interval: string): Observable<ApiResponse<BackupSettings>> {
    return this.saveAll({ autoBackupInterval: interval }).pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.autoBackup ?? { interval, options: [] },
      })),
    );
  }

  getLogSettings(): Observable<ApiResponse<LogSettings>> {
    return this.getAll().pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.logLevel ?? { level: 'info', options: [] },
      })),
    );
  }

  setLogLevel(level: string): Observable<ApiResponse<LogSettings>> {
    return this.saveAll({ logLevel: level }).pipe(
      map(r => ({ code: r.code, msg: r.msg, data: r.data?.logLevel ?? { level, options: [] } })),
    );
  }

  getProxySettings(): Observable<ApiResponse<ProxySettings>> {
    return this.getAll().pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.proxy ?? { githubProxy: '', HTTP_PROXY: '', HTTPS_PROXY: '', NO_PROXY: '' },
      })),
    );
  }

  setProxySettings(settings: ProxySettings): Observable<ApiResponse<ProxySettings>> {
    return this.saveAll({ proxy: settings }).pipe(
      map(r => ({
        code: r.code,
        msg: r.msg,
        data: r.data?.proxy ?? settings,
      })),
    );
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
