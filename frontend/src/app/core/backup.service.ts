import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface BackupSettings { retention: number; }
@Injectable({ providedIn: 'root' })
export class BackupService {
  private readonly http = inject(HttpClient);
  list(): Observable<ApiResponse<string[]>> { return this.http.get<ApiResponse<string[]>>('/api/container/listBackups'); }
  settings(): Observable<ApiResponse<BackupSettings>> { return this.http.get<ApiResponse<BackupSettings>>('/api/container/backup-settings'); }
  updateSettings(retention: number): Observable<ApiResponse<BackupSettings>> { return this.http.put<ApiResponse<BackupSettings>>('/api/container/backup-settings?retention=' + retention, {}); }
  createJson(): Observable<ApiResponse<Record<string, unknown>>> { return this.http.get<ApiResponse<Record<string, unknown>>>('/api/container/backup'); }
  createYaml(): Observable<ApiResponse<Record<string, unknown>>> { return this.http.get<ApiResponse<Record<string, unknown>>>('/api/container/backup2compose'); }
  restore(filename: string): Observable<ApiResponse<{ taskID: string }>> { return this.http.post<ApiResponse<{ taskID: string }>>('/api/container/backups/restore', { filename }); }
  remove(filename: string): Observable<ApiResponse<Record<string, unknown>>> { return this.http.delete<ApiResponse<Record<string, unknown>>>('/api/container/backups', { body: { filename } }); }
}
