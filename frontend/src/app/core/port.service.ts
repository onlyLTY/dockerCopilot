import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse, PortUsage } from './compose.service';
import { CacheStore, CacheView } from './cache-store';
import { CacheBus } from './cache-bus';

export interface PortsData { ports: PortUsage[]; conflicts: string[]; warnings: string[]; }

@Injectable({ providedIn: 'root' })
export class PortService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<PortsData>(() => this.list(), '读取端口失败');
  readonly cache: CacheView<PortsData> = this.store;

  constructor() { this.bus.register('ports', this.store); }

  ensureLoaded(): void { this.store.ensureLoaded(); }
  refresh(): void { this.store.refresh(); }

  list(): Observable<ApiResponse<PortsData>> { return this.http.get<ApiResponse<PortsData>>('/api/ports'); }
}
