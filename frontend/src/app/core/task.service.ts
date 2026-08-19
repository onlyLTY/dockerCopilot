import { Injectable, inject, signal, computed, effect } from "@angular/core";
import { HttpClient } from "@angular/common/http";
import { Subject } from "rxjs";
import { ApiResponse } from "./compose.service";
import { AuthService } from "./auth.service";
import { CacheBus, ResourceKey } from "./cache-bus";
import { ToastService } from "./toast.service";

/** 单个任务的进度快照 */
export interface TaskStep {
  message: string;
  detailMsg: string;
  stepPercentage?: number;
  progressType?: string;
  indeterminate?: boolean;
  current?: number;
  total?: number;
  startedAt: number;
  endedAt?: number;
  durationMs?: number;
  isDone: boolean;
  failed: boolean;
}

export interface TaskItem {
  taskID: string;
  title: string; // 展示用标题，如 “更新 nginx”“恢复 backup-2026-07-28”
  percentage: number; // 0-100
  message: string; // 当前阶段信息
  detailMsg: string; // 详细信息
  isDone: boolean; // 是否结束（成功或失败）
  failed: boolean; // 是否失败（以后端明确状态为准）
  canceled: boolean;
  timedOut: boolean;
  refresh: boolean; // 完成后是否需要联动刷新资源缓存
  refreshKeys?: ResourceKey[]; // 完成后需要刷新的资源；旧任务缺省按 refresh 兼容处理
  createdAt: number; // 创建时间戳，用于排序/清理
  updatedAt: number; // 最近一次进度更新时间
  startedAt?: number; // 后端记录的任务开始时间
  endedAt?: number; // 后端记录的任务结束时间
  durationMs?: number; // 后端固化的任务耗时
  resourceID?: string; // 关联资源 ID，例如容器更新对应的容器 ID
  steps: TaskStep[];
}

/** 后端 /api/progress/:taskid 的 data 结构 */
interface ProgressData {
  taskID: string;
  percentage: number;
  message: string;
  name: string;
  detailMsg: string;
  resourceID?: string;
  stepPercentage?: number;
  progressType?: string;
  indeterminate?: boolean;
  current?: number;
  total?: number;
  steps?: TaskStep[];
  isDone: boolean;
  failed?: boolean;
  canceled?: boolean;
  timedOut?: boolean;
  refresh?: boolean;
  startedAt?: number;
  endedAt?: number;
  durationMs?: number;
  updatedAt?: number;
}

interface ProgressListResponse {
  code: number;
  msg: string;
  data: ProgressData[];
}

const STORAGE_KEY = "dc-tasks";
const POLL_INTERVAL = 1500; // 批量轮询间隔（毫秒）
const DONE_KEEP = 60 * 60 * 1000; // 已完成任务保留 1 小时后可被清理
const UNKNOWN_RETRY_LIMIT = 200;
const LEGACY_REFRESH_KEYS: ResourceKey[] = [
  "containers",
  "ports",
  "images",
  "compose",
  "backups",
];

/**
 * 任务服务：后端进度持久化在 taskProgress.json，前端 localStorage 保存展示元数据。
 * - track()：发起异步操作（更新/恢复/部署）拿到 taskID 后登记，开始轮询。
 * - 使用单一定时器 + /api/progress/list 批量刷新，避免 N 任务 N 请求。
 * - isDone 后停止轮询并联动刷新相关缓存。
 */
@Injectable({ providedIn: "root" })
export class TaskService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly toast = inject(ToastService);
  private readonly auth = inject(AuthService);
  readonly completed = new Subject<{
    taskID: string;
    resourceID?: string;
    refresh: boolean;
  }>();
  readonly tasks = signal<TaskItem[]>(this.restore());
  readonly activeCount = computed(
    () => this.tasks().filter((t) => !t.isDone).length,
  );
  readonly hasActive = computed(() => this.activeCount() > 0);
  readonly activeResourceIDs = computed(
    () =>
      new Set(
        this.tasks()
          .filter((task) => !task.isDone && !!task.resourceID)
          .map((task) => task.resourceID as string),
      ),
  );
  /** 当前在弹窗中查看的任务 ID（空表示不显示进度弹窗） */
  readonly viewing = signal<string>("");
  /** 当前正在请求停止的任务 */
  readonly canceling = signal<Set<string>>(new Set());

  private batchTimer: ReturnType<typeof setTimeout> | null = null;
  private batchPolling = false;
  private unknownCounts = new Map<string, number>();
  private persistedLoaded = false;

  constructor() {
    effect(() => {
      if (this.auth.authenticated()) this.loadPersisted();
      else {
        this.persistedLoaded = false;
        this.stopBatch();
      }
    });
    if (this.tasks().some((t) => !t.isDone)) this.scheduleBatch();
  }

  /** 登记一个异步任务并开始轮询；refresh 表示完成后要联动刷新资源缓存 */
  track(
    taskID: string,
    title: string,
    refresh = false,
    resourceID = "",
    refreshKeys?: ResourceKey[],
  ): void {
    if (!taskID || !this.auth.authenticated()) return;
    const now = Date.now();
    const existing = this.tasks().find((t) => t.taskID === taskID);
    if (existing) {
      this.tasks.update((list) =>
        list.map((item) =>
          item.taskID === taskID
            ? {
                ...item,
                title: title || item.title,
                resourceID: resourceID || item.resourceID,
                refresh: refresh || item.refresh,
                refreshKeys: refreshKeys?.length
                  ? refreshKeys
                  : item.refreshKeys,
              }
            : item,
        ),
      );
      this.persist();
      this.toast.info(`任务「${title}」已创建，可前往任务页面查看详情`);
      return;
    }
    const item: TaskItem = {
      taskID,
      title,
      percentage: 0,
      message: "任务已提交",
      detailMsg: "",
      isDone: false,
      failed: false,
      canceled: false,
      timedOut: false,
      refresh,
      refreshKeys: refreshKeys?.length ? refreshKeys : undefined,
      createdAt: now,
      updatedAt: now,
      startedAt: now,
      endedAt: undefined,
      durationMs: undefined,
      resourceID,
      steps: [],
    };
    this.tasks.update((list) => [item, ...list]);
    this.persist();
    this.toast.info(`任务「${title}」已创建，可前往任务页面查看详情`);
    this.scheduleBatch(true);
  }

  private loadPersisted(): void {
    if (this.persistedLoaded) return;
    this.persistedLoaded = true;
    this.http.get<ProgressListResponse>("/api/progress/list").subscribe({
      next: (response) => {
        if (response.code !== 200 || !Array.isArray(response.data)) {
          this.persistedLoaded = false;
          return;
        }
        const now = Date.now();
        const local = new Map(this.tasks().map((item) => [item.taskID, item]));
        const merged = response.data.map((progress) => {
          const existing = local.get(progress.taskID);
          const serverUpdated =
            typeof progress.updatedAt === "number" && progress.updatedAt > 0
              ? progress.updatedAt
              : now;
          if (existing) {
            local.delete(progress.taskID);
            const serverDone = progress.isDone;
            const activeDone = existing.isDone;
            const hasServerSteps =
              Array.isArray(progress.steps) && progress.steps.length > 0;
            const newer = serverUpdated >= (existing.updatedAt || 0);
            const acceptServerSnapshot = newer || serverDone || hasServerSteps;
            return {
              ...existing,
              percentage: Math.max(
                existing.percentage,
                progress.percentage || 0,
              ),
              message:
                acceptServerSnapshot && progress.message
                  ? progress.message
                  : existing.message,
              detailMsg:
                acceptServerSnapshot && progress.detailMsg
                  ? progress.detailMsg
                  : existing.detailMsg,
              resourceID: progress.resourceID || existing.resourceID,
              steps: hasServerSteps ? progress.steps! : existing.steps || [],
              isDone: activeDone || serverDone,
              canceled: existing.canceled || !!progress.canceled,
              timedOut: existing.timedOut || !!progress.timedOut,
              failed:
                existing.failed ||
                !!progress.failed ||
                (serverDone && progress.percentage < 100),
              refresh:
                existing.refresh || !!progress.refresh || !!progress.resourceID,
              startedAt:
                progress.startedAt && progress.startedAt > 0
                  ? progress.startedAt
                  : existing.startedAt,
              endedAt:
                progress.endedAt && progress.endedAt > 0
                  ? progress.endedAt
                  : existing.endedAt,
              durationMs:
                typeof progress.durationMs === "number" &&
                progress.durationMs >= 0
                  ? progress.durationMs
                  : existing.durationMs,
              updatedAt: Math.max(existing.updatedAt || 0, serverUpdated),
            };
          }
          return {
            taskID: progress.taskID,
            title: progress.name || progress.taskID,
            percentage: progress.percentage,
            message: progress.message || "任务已恢复",
            detailMsg: progress.detailMsg || "",
            resourceID: progress.resourceID,
            steps: progress.steps || [],
            isDone: progress.isDone,
            canceled: progress.canceled ?? false,
            timedOut: progress.timedOut ?? false,
            failed:
              progress.failed ??
              (progress.isDone &&
                (progress.percentage < 100 ||
                  /失败|错误|中断|拒绝|超时|error|fail|timeout/i.test(
                    progress.message || "",
                  ))),
            refresh: progress.refresh ?? !!progress.resourceID,
            startedAt:
              progress.startedAt && progress.startedAt > 0
                ? progress.startedAt
                : undefined,
            endedAt:
              progress.endedAt && progress.endedAt > 0
                ? progress.endedAt
                : undefined,
            durationMs:
              typeof progress.durationMs === "number" && progress.durationMs >= 0
                ? progress.durationMs
                : undefined,
            createdAt: serverUpdated,
            updatedAt: serverUpdated,
          };
        });
        const restoredDone = this.tasks().filter(
          (item) =>
            !item.isDone &&
            merged.some((next) => next.taskID === item.taskID && next.isDone),
        );
        this.tasks.set([...merged, ...local.values()]);
        for (const item of restoredDone) {
          if (item.refresh) this.bus.refresh(this.taskRefreshKeys(item));
          this.completed.next({
            taskID: item.taskID,
            resourceID: item.resourceID,
            refresh: item.refresh,
          });
        }
        this.persist();
        if (this.tasks().some((item) => !item.isDone)) this.scheduleBatch(true);
      },
      error: () => {
        this.persistedLoaded = false;
      },
    });
  }

  /** 打开某个任务的进度弹窗 */
  view(taskID: string): void {
    this.viewing.set(taskID);
  }
  /** 关闭进度弹窗（任务继续在后台轮询） */
  closeView(): void {
    this.viewing.set("");
  }

  /** 删除单个任务；进行中的任务不能删除，避免把执行中的任务伪装成已取消。 */
  remove(taskID: string): void {
    const item = this.tasks().find((t) => t.taskID === taskID);
    if (item && !item.isDone) {
      this.toast.info("任务进行中，完成后才能删除");
      return;
    }
    this.canceling.update((set) => {
      const next = new Set(set);
      next.delete(taskID);
      return next;
    });
    this.unknownCounts.delete(taskID);
    this.tasks.update((list) => list.filter((t) => t.taskID !== taskID));
    if (this.viewing() === taskID) this.viewing.set("");
    this.persist();
    if (!this.tasks().some((t) => !t.isDone)) this.stopBatch();
    this.http.delete("/api/progress/" + encodeURIComponent(taskID)).subscribe({
      error: () =>
        this.toast.error(
          "删除任务失败",
          "无法删除后端记录，刷新后可能重新出现",
        ),
    });
  }

  /** 请求停止单个进行中的任务；记录仍保留，等待轮询确认取消终态。 */
  cancel(taskID: string): Promise<boolean> {
    const item = this.tasks().find((t) => t.taskID === taskID);
    if (!item || item.isDone) {
      this.toast.info("任务已结束", "已结束的任务无需停止");
      return Promise.resolve(false);
    }
    if (this.canceling().has(taskID)) return Promise.resolve(true);
    this.canceling.update((set) => new Set(set).add(taskID));
    return new Promise((resolve) => {
      this.http
        .post<ApiResponse<unknown>>(
          "/api/progress/" + encodeURIComponent(taskID) + "/cancel",
          {},
        )
        .subscribe({
          next: (response) => {
            if (response.code >= 200 && response.code < 300) {
              this.toast.info("正在停止任务", "等待后端确认任务已取消");
              this.scheduleBatch(true);
              resolve(true);
            } else {
              this.canceling.update((set) => {
                const next = new Set(set);
                next.delete(taskID);
                return next;
              });
              this.toast.error(
                "停止任务失败",
                response.msg || "任务当前无法停止",
              );
              resolve(false);
            }
          },
          error: (error) => {
            this.canceling.update((set) => {
              const next = new Set(set);
              next.delete(taskID);
              return next;
            });
            this.toast.error(
              "停止任务失败",
              error?.error?.msg || "无法连接服务端",
            );
            resolve(false);
          },
        });
    });
  }

  /** 请求停止所有进行中的任务，不删除任务记录。 */
  async cancelAllActive(): Promise<number> {
    const active = this.tasks()
      .filter((t) => !t.isDone)
      .map((t) => t.taskID);
    const results = await Promise.all(
      active.map((taskID) => this.cancel(taskID)),
    );
    return results.filter(Boolean).length;
  }

  /** 清空全部：先清理已结束记录，再请求停止进行中任务。 */
  async clearAll(): Promise<void> {
    this.clearDone();
    const active = this.tasks().filter((t) => !t.isDone);
    if (!active.length) return;
    const count = await this.cancelAllActive();
    if (count)
      this.toast.info(
        "已请求停止任务",
        `已请求停止 ${count} 个进行中任务，取消确认后仍可清理记录`,
      );
  }

  /** 清除已完成的任务，保留进行中的 */
  clearDone(): void {
    this.tasks.update((list) => list.filter((t) => !t.isDone));
    if (
      this.viewing() &&
      !this.tasks().some((t) => t.taskID === this.viewing())
    )
      this.viewing.set("");
    this.persist();
    if (!this.tasks().some((t) => !t.isDone)) this.stopBatch();
    // 同步清空后端已完成记录
    this.http.delete("/api/progress/clear?doneOnly=true").subscribe({
      error: () =>
        this.toast.error(
          "清除已完成失败",
          "无法清除后端记录，刷新后可能重新出现",
        ),
    });
  }

  private scheduleBatch(immediate = false): void {
    if (this.batchPolling) return;
    if (this.batchTimer) {
      if (!immediate) return;
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }
    const delay = immediate ? 0 : POLL_INTERVAL;
    this.batchTimer = setTimeout(() => {
      this.batchTimer = null;
      this.pollBatch();
    }, delay);
  }

  private stopBatch(): void {
    if (this.batchTimer) {
      clearTimeout(this.batchTimer);
      this.batchTimer = null;
    }
    this.batchPolling = false;
  }

  private pollBatch(): void {
    const active = this.tasks().filter((t) => !t.isDone);
    if (!active.length) {
      this.stopBatch();
      return;
    }
    if (this.batchPolling) return;
    this.batchPolling = true;

    const finish = () => {
      this.batchPolling = false;
      if (this.tasks().some((t) => !t.isDone)) this.scheduleBatch();
    };

    this.http.get<ProgressListResponse>("/api/progress/list").subscribe({
      next: (response) => {
        if (response.code === 200 && Array.isArray(response.data)) {
          const byId = new Map(response.data.map((p) => [p.taskID, p]));
          const missing: string[] = [];
          for (const item of active) {
            const progress = byId.get(item.taskID);
            if (progress) {
              this.unknownCounts.delete(item.taskID);
              this.apply(item.taskID, progress);
            } else {
              missing.push(item.taskID);
            }
          }
          if (missing.length) {
            this.pollActiveIndividually(missing, finish);
            return;
          }
        } else {
          // list 失败时退化为逐个拉取（最多 3 个并发感，串行即可保持简单）
          this.pollActiveIndividually(
            active.map((t) => t.taskID),
            finish,
          );
          return;
        }
        finish();
      },
      error: () => {
        this.pollActiveIndividually(
          active.map((t) => t.taskID),
          finish,
        );
      },
    });
  }

  private pollActiveIndividually(ids: string[], done: () => void): void {
    let pending = ids.length;
    if (!pending) {
      done();
      return;
    }
    const step = () => {
      pending--;
      if (pending <= 0) done();
    };
    for (const taskID of ids) {
      this.http
        .get<ApiResponse<ProgressData>>(
          "/api/progress/" + encodeURIComponent(taskID),
        )
        .subscribe({
          next: (r) => {
            if (r.code !== 200 || !r.data) this.markUnknown(taskID, r.msg);
            else {
              this.unknownCounts.delete(taskID);
              this.apply(taskID, r.data);
            }
            step();
          },
          error: (e) => {
            this.markUnknown(
              taskID,
              e?.error?.msg || e?.message || "任务进度查询失败",
            );
            step();
          },
        });
    }
  }

  private taskRefreshKeys(task: Pick<TaskItem, "refresh" | "refreshKeys">): ResourceKey[] {
    return task.refreshKeys?.length
      ? task.refreshKeys
      : task.refresh
        ? LEGACY_REFRESH_KEYS
        : [];
  }

  private apply(taskID: string, d: ProgressData): void {
    let doneNow = false;
    let refreshNeeded = false;
    let refreshKeys: ResourceKey[] = [];
    let failedNow = false;
    let canceledNow = false;
    let timedOutNow = false;
    let failureTitle = "";
    let failureMessage = "";
    this.tasks.update((list) =>
      list.map((t) => {
        if (t.taskID !== taskID || t.isDone) return t;
        const updatedAt =
          typeof d.updatedAt === "number" && d.updatedAt > 0
            ? d.updatedAt
            : Date.now();
        const hasServerSteps = Array.isArray(d.steps) && d.steps.length > 0;
        if (
          updatedAt < (t.updatedAt || 0) &&
          !d.isDone &&
          !hasServerSteps
        )
          return t;
        const failed =
          d.failed ??
          (d.isDone &&
            (d.percentage < 100 ||
              /失败|错误|中断|拒绝|超时|error|fail|timeout/i.test(
                d.message || "",
              )));
        if (d.isDone) {
          doneNow = true;
          refreshNeeded = t.refresh;
          refreshKeys = this.taskRefreshKeys(t);
          failedNow = failed;
          canceledNow = !!d.canceled;
          timedOutNow = !!d.timedOut;
          failureTitle = d.name || t.title;
          failureMessage = d.detailMsg || d.message || "任务执行失败";
        }
        return {
          ...t,
          percentage: Math.max(t.percentage, d.percentage || 0),
          message: d.message || t.message,
          detailMsg: d.detailMsg || t.detailMsg,
          steps: hasServerSteps ? d.steps! : t.steps || [],
          isDone: d.isDone,
          canceled: !!d.canceled,
          timedOut: !!d.timedOut,
          failed,
          startedAt:
            typeof d.startedAt === "number" && d.startedAt > 0
              ? d.startedAt
              : t.startedAt,
          endedAt:
            typeof d.endedAt === "number" && d.endedAt > 0
              ? d.endedAt
              : t.endedAt,
          durationMs:
            typeof d.durationMs === "number" && d.durationMs >= 0
              ? d.durationMs
              : t.durationMs,
          updatedAt,
        };
      }),
    );
    if (doneNow) {
      this.canceling.update((set) => {
        const next = new Set(set);
        next.delete(taskID);
        return next;
      });
      this.unknownCounts.delete(taskID);
      if (canceledNow)
        this.toast.info(
          "任务已取消",
          failureMessage || "任务已停止；已执行的 Docker 操作不会自动回滚",
        );
      else if (timedOutNow)
        this.toast.error(
          "任务超时",
          failureMessage ||
            "任务超过设定时间，已执行的 Docker 操作不会自动回滚",
        );
      else if (failedNow)
        this.toast.error(failureTitle || "任务失败", failureMessage);
      else this.toast.success("任务完成", this.summaryMessage(failureMessage, failureTitle));
      if (refreshNeeded) this.bus.refresh(refreshKeys);
      this.completed.next({
        taskID,
        resourceID: this.tasks().find((t) => t.taskID === taskID)?.resourceID,
        refresh: refreshNeeded,
      });
    }
    this.persist();
  }

  private summaryMessage(detail: string, fallback: string): string {
    const firstLine = detail
      .split(/\r?\n/)
      .map((line) => line.trim())
      .find((line) => line.length);
    return firstLine || fallback || "异步任务已完成";
  }

  private markUnknown(taskID: string, msg: string): void {
    const attempts = (this.unknownCounts.get(taskID) || 0) + 1;
    this.unknownCounts.set(taskID, attempts);
    if (attempts < UNKNOWN_RETRY_LIMIT) return;
    let changed = false;
    let refreshNeeded = false;
    let refreshKeys: ResourceKey[] = [];
    this.tasks.update((list) =>
      list.map((t) => {
        if (t.taskID !== taskID || t.isDone) return t;
        changed = true;
        refreshNeeded = t.refresh;
        refreshKeys = this.taskRefreshKeys(t);
        return {
          ...t,
          isDone: true,
          failed: true,
          message: msg || "任务不存在或已过期",
          detailMsg: msg || "",
          updatedAt: Date.now(),
        };
      }),
    );
    this.unknownCounts.delete(taskID);
    if (changed) {
      this.canceling.update((set) => {
        const next = new Set(set);
        next.delete(taskID);
        return next;
      });
      this.toast.error("任务失败", msg || "任务不存在或已过期");
      if (refreshNeeded) this.bus.refresh(refreshKeys);
      this.completed.next({
        taskID,
        resourceID: this.tasks().find((t) => t.taskID === taskID)?.resourceID,
        refresh: refreshNeeded,
      });
    }
    this.persist();
  }

  private persist(): void {
    try {
      const now = Date.now();
      const keep = this.tasks().filter(
        (t) => !t.isDone || now - t.updatedAt < DONE_KEEP,
      );
      localStorage.setItem(STORAGE_KEY, JSON.stringify(keep));
    } catch {
      /* localStorage 不可用则忽略 */
    }
  }

  private restore(): TaskItem[] {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      const list = JSON.parse(raw) as TaskItem[];
      return Array.isArray(list) ? list : [];
    } catch {
      return [];
    }
  }
}
