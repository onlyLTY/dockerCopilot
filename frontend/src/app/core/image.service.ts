import { Injectable, inject } from "@angular/core";
import {
  HttpClient,
  HttpEventType,
  HttpRequest,
} from "@angular/common/http";
import { Observable, tap } from "rxjs";
import { ApiResponse } from "./compose.service";
import { CacheStore, CacheView } from "./cache-store";
import { CacheBus } from "./cache-bus";
import { actionErrorMessage } from "./run-action";
import { TaskService } from "./task.service";

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

export interface ImageImportSubmitResult {
  taskID: string;
  filename: string;
}

/** 生成后端接受的任务编号（[A-Za-z0-9_-]{8,64}）；非 HTTPS 环境下 randomUUID 可能不可用。 */
export function newImportTaskID(): string {
  const cryptoObj = globalThis.crypto;
  if (cryptoObj?.randomUUID) return cryptoObj.randomUUID();
  const bytes = new Uint8Array(16);
  if (cryptoObj?.getRandomValues) cryptoObj.getRandomValues(bytes);
  else for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

@Injectable({ providedIn: "root" })
export class ImageService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly tasks = inject(TaskService);
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
   * 上传 docker save 导出的镜像包；后端收完上传后立即返回，docker load 由后台任务执行。
   * 订阅保存在 root 服务里，离开镜像页不会中断上传，进度统一在任务页查看。
   * taskID 由前端生成，上传开始前就能登记任务。
   */
  startImport(file: File, taskID: string): void {
    this.tasks.track(taskID, `导入镜像 ${file.name}`, true, "", ["images"]);
    let uploaded = false;
    const req = new HttpRequest<File>(
      "POST",
      `/api/images/import?filename=${encodeURIComponent(file.name)}&taskID=${encodeURIComponent(taskID)}`,
      file,
      { reportProgress: true },
    );
    this.http
      .request<ApiResponse<ImageImportSubmitResult>>(req)
      .subscribe({
        next: (event) => {
          if (event.type === HttpEventType.UploadProgress) {
            uploaded = !!event.total && event.loaded >= event.total;
            return;
          }
          if (event.type !== HttpEventType.Response) return;
          const body = event.body as ApiResponse<ImageImportSubmitResult>;
          if (body?.code !== 200) {
            this.tasks.fail(taskID, body?.msg || "导入失败");
          }
        },
        error: (e) => {
          // 上传完成后的连接错误不代表任务失败：后端可能已在执行导入
          const status = (e as { status?: number } | null)?.status;
          if (uploaded || status === 0) {
            this.tasks.confirmOrFail(
              taskID,
              status === 0 ? "无法连接服务器" : "导入请求中断",
            );
            return;
          }
          this.tasks.fail(
            taskID,
            status === 413
              ? "镜像包超过服务端允许的大小限制"
              : actionErrorMessage(e, "导入失败"),
          );
        },
      });
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
