import { Injectable, inject } from "@angular/core";
import { HttpClient, HttpHeaders } from "@angular/common/http";
import { Observable, tap } from "rxjs";
import { ApiResponse } from "./compose.service";
import { CacheStore, CacheView } from "./cache-store";
import { CacheBus } from "./cache-bus";
import { SettingsService } from "./settings.service";

export interface BackupSettings {
  retention: number;
}

@Injectable({ providedIn: "root" })
export class BackupService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly store = new CacheStore<string[]>(
    () => this.list(),
    "读取备份失败",
    { ttlMs: 0 },
  );
  private readonly settingsStore = new CacheStore<BackupSettings>(
    () => this.settings(),
    "读取备份设置失败",
    { ttlMs: 0 },
  );
  private readonly runtimeSettings = inject(SettingsService);
  readonly cache: CacheView<string[]> = this.store;
  readonly settingsCache: CacheView<BackupSettings> = this.settingsStore;

  constructor() {
    this.bus.register("backups", this.store);
  }

  ensureLoaded(): void {
    this.store.ensureLoaded();
    this.settingsStore.ensureLoaded();
  }
  refresh(): void {
    this.store.refresh();
  }

  list(): Observable<ApiResponse<string[]>> {
    return this.http.get<ApiResponse<string[]>>("/api/container/listBackups");
  }
  settings(): Observable<ApiResponse<BackupSettings>> {
    return this.http.get<ApiResponse<BackupSettings>>(
      "/api/container/backup-settings",
    );
  }
  updateSettings(retention: number): Observable<ApiResponse<BackupSettings>> {
    return this.http
      .put<ApiResponse<BackupSettings>>(
        "/api/container/backup-settings?retention=" + retention,
        {},
      )
      .pipe(
        tap((r) => {
          if (r.code === 200) {
            this.runtimeSettings.setRuntime({ retention });
            this.settingsStore.refresh();
            this.store.refresh();
          }
        }),
      );
  }
  createJson(): Observable<ApiResponse<{ taskID: string }>> {
    return this.http.get<ApiResponse<{ taskID: string }>>(
      "/api/container/backup",
    );
  }
  createYaml(): Observable<ApiResponse<{ taskID: string }>> {
    return this.http.get<ApiResponse<{ taskID: string }>>(
      "/api/container/backup2compose",
    );
  }
  restore(filename: string): Observable<ApiResponse<{ taskID: string }>> {
    return this.http
      .post<ApiResponse<{ taskID: string }>>("/api/container/backups/restore", {
        filename,
      })
      .pipe(
        tap((r) => {
          if (r.code === 200)
            this.bus.invalidate(["containers", "ports", "images"]);
        }),
      );
  }
  remove(filename: string): Observable<ApiResponse<{ taskID: string }>> {
    return this.http.delete<ApiResponse<{ taskID: string }>>(
      "/api/container/backups",
      {
        body: { filename },
        headers: new HttpHeaders({ "Content-Type": "application/json" }),
      },
    );
  }

  private onCreate(r: ApiResponse<unknown>): void {
    if (r.code === 200) this.store.refresh();
  }
}
