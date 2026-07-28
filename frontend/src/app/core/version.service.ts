import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface VersionInfo { version: string; buildDate: string; }

@Injectable({ providedIn: 'root' })
export class VersionService {
  private readonly http = inject(HttpClient);
  local(): Observable<ApiResponse<VersionInfo>> { return this.http.get<ApiResponse<VersionInfo>>('/api/version?type=local'); }
}
