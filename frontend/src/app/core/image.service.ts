import { Injectable, inject } from "@angular/core";
import {
  HttpClient,
  HttpEventType,
  HttpRequest,
  HttpResponse,
} from "@angular/common/http";
import { Observable, filter, map, tap } from "rxjs";
import { ApiResponse } from "./compose.service";
import { CacheStore, CacheView } from "./cache-store";
import { CacheBus } from "./cache-bus";

export interface ImageRow {
  id: string;
  name: string;
  tag: string;
  repoTags: string[];
  size: string;
  inUsed: boolean;
  createTime: string;
}

export interface ImageCleanupResult {
  deleted: number;
  skipped: number;
  errors: string[];
}

export interface ImageImportResult {
  loaded: string[];
  count: number;
}

@Injectable({ providedIn: "root" })
export class ImageService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<ImageRow[]>(
    () => this.list(),
    "读取镜像失败",
    { ttlMs: 0 },
  );
  readonly cache: CacheView<ImageRow[]> = this.store;

  constructor() {
    this.bus.register("images", this.store);
  }

  ensureLoaded(): void {
    this.store.ensureLoaded();
  }
  refresh(): void {
    this.store.refresh();
  }

  list(): Observable<ApiResponse<ImageRow[]>> {
    return this.http.get<ApiResponse<ImageRow[]>>("/api/images");
  }
  remove(
    id: string,
    force = false,
    repoTag?: string,
  ): Observable<ApiResponse<Record<string, unknown>>> {
    const params = new URLSearchParams({ force: String(force) });
    if (!force && repoTag) params.set("repoTag", repoTag);
    return this.done(
      this.http.delete<ApiResponse<Record<string, unknown>>>(
        `/api/image/${encodeURIComponent(id)}?${params.toString()}`,
      ),
    );
  }
  cleanup(
    kind: "untagged" | "unused",
  ): Observable<ApiResponse<{ taskID: string }>> {
    return this.http.post<ApiResponse<{ taskID: string }>>(
      "/api/images/prune?kind=" + kind,
      {},
    );
  }

  /**
   * 导入 docker save 导出的镜像包。
   * 直接上传原始文件流（不套 multipart），大文件不占内存；onProgress 回报上传进度。
   */
  import(
    file: File,
    onProgress: (percent: number) => void,
  ): Observable<ApiResponse<ImageImportResult>> {
    const req = new HttpRequest<File>(
      "POST",
      "/api/images/import?filename=" + encodeURIComponent(file.name),
      file,
      { reportProgress: true },
    );
    return this.http.request<ApiResponse<ImageImportResult>>(req).pipe(
      tap((event) => {
        if (event.type === HttpEventType.UploadProgress) {
          const total = event.total || file.size;
          onProgress(total > 0 ? Math.round((event.loaded / total) * 100) : 0);
        }
      }),
      filter(
        (event): event is HttpResponse<ApiResponse<ImageImportResult>> =>
          event instanceof HttpResponse,
      ),
      map((event) => event.body as ApiResponse<ImageImportResult>),
      tap((r) => {
        if (r.code === 200) {
          this.store.refresh();
          this.bus.invalidate(["containers"]);
        }
      }),
    );
  }

  /** 删除/清理成功后：刷新镜像自身缓存，并让容器缓存失效（更新标记可能变化） */
  private done<T>(obs: Observable<ApiResponse<T>>): Observable<ApiResponse<T>> {
    return obs.pipe(
      tap((r) => {
        if (r.code === 200) {
          this.store.refresh();
          this.bus.invalidate(["containers"]);
        }
      }),
    );
  }
}
