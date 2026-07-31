import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, forkJoin, tap } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface UpdateSettings { interval: string; options: string[]; }
export interface BackupSettings { interval: string; options: string[]; }
export interface LogSettings { level: string; options: string[]; }
export interface ProxySettings { githubProxy: string; HTTP_PROXY: string; HTTPS_PROXY: string; NO_PROXY: string; }

export interface RuntimeSettings {
  updateInterval: string;
  backupInterval: string;
  logLevel: string;
  retention: number;
}

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly http = inject(HttpClient);
  readonly runtime = signal<RuntimeSettings>({ updateInterval: '', backupInterval: '', logLevel: 'info', retention: 10 });
  private loaded = false;

  constructor() { this.load(); }

  load(): void {
    if (this.loaded) return;
    this.loaded = true;
    forkJoin({
      update: this.getUpdateSettings(),
      backup: this.getBackupSettings(),
      log: this.getLogSettings(),
      retention: this.http.get<ApiResponse<{ retention: number }>>('/api/container/backup-settings'),
    }).subscribe({
      next: result => this.runtime.set({
        updateInterval: result.update.data.interval,
        backupInterval: result.backup.data.interval,
        logLevel: result.log.data.level,
        retention: result.retention.data.retention,
      }),
      error: () => { this.loaded = false; },
    });
  }

  getUpdateSettings(): Observable<ApiResponse<UpdateSettings>> { return this.http.get<ApiResponse<UpdateSettings>>('/api/settings/update-check'); }
  setRuntime(value: Partial<RuntimeSettings>): void { this.runtime.update(current => ({ ...current, ...value })); }
  setUpdateInterval(interval: string): Observable<ApiResponse<UpdateSettings>> { return this.http.put<ApiResponse<UpdateSettings>>('/api/settings/update-check', { interval }).pipe(tap(r => { if (r.code === 200) this.setRuntime({ updateInterval: interval }); })); }
  getBackupSettings(): Observable<ApiResponse<BackupSettings>> { return this.http.get<ApiResponse<BackupSettings>>('/api/settings/auto-backup'); }
  setBackupInterval(interval: string): Observable<ApiResponse<BackupSettings>> { return this.http.put<ApiResponse<BackupSettings>>('/api/settings/auto-backup', { interval }).pipe(tap(r => { if (r.code === 200) this.setRuntime({ backupInterval: interval }); })); }
  getLogSettings(): Observable<ApiResponse<LogSettings>> { return this.http.get<ApiResponse<LogSettings>>('/api/settings/log-level'); }
  setLogLevel(level: string): Observable<ApiResponse<LogSettings>> { return this.http.put<ApiResponse<LogSettings>>('/api/settings/log-level', { level }).pipe(tap(r => { if (r.code === 200) this.setRuntime({ logLevel: level }); })); }
  getProxySettings(): Observable<ApiResponse<ProxySettings>> { return this.http.get<ApiResponse<ProxySettings>>('/api/settings/proxy'); }
  setProxySettings(settings: ProxySettings): Observable<ApiResponse<ProxySettings>> { return this.http.put<ApiResponse<ProxySettings>>('/api/settings/proxy', settings); }
}
