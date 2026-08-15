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
  private cachedAt = 0;
  private cacheVersion = 0;
  private settingsRequest$: Observable<ApiResponse<AppSettings>> | null = null;
  private readonly cacheTtlMs = 30_000;
  private operationID = '';
  readonly daemonOperation = signal<DaemonRestartOperation | null>(null);
  private daemonPollTimer: ReturnType<typeof setTimeout> | null = null;
  private daemonPollStartedAt = 0;

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
    return this.http.post<ApiResponse<DaemonRestartOperation>>('/api/daemon/restart', {}).pipe(
      tap(response => {
        if (response.code === 202 && response.data) {
          this.operationID = response.data.operationID;
          this.daemonOperation.set(response.data);
          this.daemonPollStartedAt = Date.now();
          this.pollDaemonOperation();
        }
      }),
    );
  }

  resumeDaemonOperation(): void {
    if (this.operationID && !this.daemonOperation()?.status?.match(/succeeded|failed/)) {
      this.daemonPollStartedAt ||= Date.now();
      this.pollDaemonOperation();
    }
  }

  private pollDaemonOperation(): void {
    if (!this.operationID || Date.now() - this.daemonPollStartedAt > 120_000) return;
    this.getDaemonOperation(this.operationID).subscribe({
      next: response => {
        if (response.code !== 200 || !response.data) return;
        this.daemonOperation.set(response.data);
        if (response.data.status === 'restarting') {
          this.daemonPollTimer = setTimeout(() => this.pollDaemonOperation(), 1500);
        }
      },
      error: () => {
        this.daemonPollTimer = setTimeout(() => this.pollDaemonOperation(), 2000);
      },
    });
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
