import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface UpdateSettings { interval: string; options: string[]; }
export interface BackupSettings { interval: string; options: string[]; }

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly http = inject(HttpClient);
  getUpdateSettings(): Observable<ApiResponse<UpdateSettings>> { return this.http.get<ApiResponse<UpdateSettings>>('/api/settings/update-check'); }
  setUpdateInterval(interval: string): Observable<ApiResponse<UpdateSettings>> { return this.http.put<ApiResponse<UpdateSettings>>('/api/settings/update-check', { interval }); }
  getBackupSettings(): Observable<ApiResponse<BackupSettings>> { return this.http.get<ApiResponse<BackupSettings>>('/api/settings/auto-backup'); }
  setBackupInterval(interval: string): Observable<ApiResponse<BackupSettings>> { return this.http.put<ApiResponse<BackupSettings>>('/api/settings/auto-backup', { interval }); }
}
