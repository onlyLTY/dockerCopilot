import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse, PortUsage } from './compose.service';

export interface PortsData { ports: PortUsage[]; conflicts: string[]; warnings: string[]; }

@Injectable({ providedIn: 'root' })
export class PortService {
  private readonly http = inject(HttpClient);
  list(): Observable<ApiResponse<PortsData>> { return this.http.get<ApiResponse<PortsData>>('/api/ports'); }
}
