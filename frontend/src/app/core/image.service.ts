import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface ImageRow { id: string; name: string; tag: string; size: string; inUsed: boolean; createTime: string; }

export interface ImageCleanupResult { deleted: number; skipped: number; errors: string[]; }

@Injectable({ providedIn: 'root' })
export class ImageService {
  private readonly http = inject(HttpClient);
  list(): Observable<ApiResponse<ImageRow[]>> { return this.http.get<ApiResponse<ImageRow[]>>('/api/images'); }
  remove(id: string, force = false): Observable<ApiResponse<Record<string, unknown>>> {
    return this.http.delete<ApiResponse<Record<string, unknown>>>(`/api/image/${encodeURIComponent(id)}?force=${force}`);
  }
  cleanup(kind: 'untagged' | 'unused'): Observable<ApiResponse<ImageCleanupResult>> {
    return this.http.post<ApiResponse<ImageCleanupResult>>('/api/images/prune?kind=' + kind, {});
  }
}
