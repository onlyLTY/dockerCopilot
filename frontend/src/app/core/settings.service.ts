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
  HTTP_PROXY: string;
  HTTPS_PROXY: string;
  NO_PROXY: string;
}

export interface DaemonProxyDraft {
  httpProxy: string;
  httpsProxy: string;
  noProxy: string;
}

export interface DaemonProxySettings {
  helperEnabled: boolean;
  fileExists: boolean;
  writable: boolean;
  hash: string;
  fileProxy: DaemonProxyDraft;
  effectiveProxy: DaemonProxyDraft;
  restartRequired: boolean;
  message?: string;
}

export interface DaemonRestartOperation {
  operationID: string;
  status: string;
  message?: string;
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
  daemonProxyDraft: DaemonProxyDraft;
  daemonProxyDraftConfigured: boolean;
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
  daemonProxyDraft?: DaemonProxyDraft;
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
  private settingsRequest$: Observable<ApiResponse<AppSettings>> | null = null;

  constructor() {
    this.load();
  }

  getAll(): Observable<ApiResponse<AppSettings>> {
    if (this.cachedSnapshot) return of({ code: 200, msg: '', data: this.cachedSnapshot });
    if (!this.settingsRequest$) {
      this.settingsRequest$ = this.http.get<ApiResponse<AppSettings>>('/api/settings').pipe(
        tap(result => {
          if (result.code === 200 && result.data) {
            this.cachedSnapshot = result.data;
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

  saveAll(body: AppSettingsUpdate): Observable<ApiResponse<AppSettings>> {
    return this.http.put<ApiResponse<AppSettings>>('/api/settings', body).pipe(
      tap(r => {
        if (r.code === 200 && r.data) {
          this.cachedSnapshot = r.data;
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

  getDaemonProxy(): Observable<ApiResponse<DaemonProxySettings>> {
    return this.http.get<ApiResponse<DaemonProxySettings>>('/api/settings/proxy/daemon');
  }

  applyDaemonProxy(settings: {
    httpProxy: string;
    httpsProxy: string;
    noProxy: string;
    hash: string;
  }): Observable<ApiResponse<{ status: DaemonProxySettings; backupCreated: boolean }>> {
    return this.http.post<ApiResponse<{ status: DaemonProxySettings; backupCreated: boolean }>>(
      '/api/settings/proxy/daemon',
      settings,
    );
  }

  restartDaemon(): Observable<ApiResponse<DaemonRestartOperation>> {
    return this.http.post<ApiResponse<DaemonRestartOperation>>('/api/daemon/restart', {});
  }

  getDaemonOperation(operationID: string): Observable<ApiResponse<DaemonRestartOperation>> {
    return this.http.get<ApiResponse<DaemonRestartOperation>>(
      '/api/daemon/restart/' + encodeURIComponent(operationID),
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
