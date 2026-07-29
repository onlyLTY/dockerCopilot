import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Subject } from 'rxjs';
import { ApiResponse } from './compose.service';
import { CacheBus } from './cache-bus';
import { ToastService } from './toast.service';

/** 单个任务的进度快照 */
export interface TaskItem {
  taskID: string;
  title: string;          // 展示用标题，如 “更新 nginx”“恢复 backup-2026-07-28”
  percentage: number;     // 0-100
  message: string;        // 当前阶段信息
  detailMsg: string;      // 详细信息
  isDone: boolean;        // 是否结束（成功或失败）
  failed: boolean;        // 是否失败（isDone 且百分比未达成或消息含失败）
  refresh: boolean;       // 完成后是否需要联动刷新资源缓存
  createdAt: number;      // 创建时间戳，用于排序/清理
  updatedAt: number;      // 最近一次进度更新时间
  resourceID?: string;    // 关联资源 ID，例如容器更新对应的容器 ID
}

/** 后端 /api/progress/:taskid 的 data 结构 */
interface ProgressData {
  taskID: string;
  percentage: number;
  message: string;
  name: string;
  detailMsg: string;
  isDone: boolean;
}

const STORAGE_KEY = 'dc-tasks';
const POLL_INTERVAL = 1500;   // 轮询间隔（毫秒）
const DONE_KEEP = 60 * 60 * 1000; // 已完成任务保留 1 小时后可被清理

/**
 * 任务服务：后端进度存于内存且只提供单任务查询，因此任务列表在前端维护。
 * - track()：发起异步操作（更新/恢复/部署）拿到 taskID 后登记，开始轮询。
 * - 轮询 /api/progress/:taskid 更新进度；isDone 后停止轮询并联动刷新相关缓存。
 * - 列表持久化到 localStorage，刷新页面后仍可看到，未完成的会恢复轮询。
 */
@Injectable({ providedIn: 'root' })
export class TaskService {
  private readonly http = inject(HttpClient);
  private readonly bus = inject(CacheBus);
  private readonly toast = inject(ToastService);
  readonly completed = new Subject<{ taskID: string; resourceID?: string; refresh: boolean }>();
  readonly tasks = signal<TaskItem[]>(this.restore());
  readonly activeCount = computed(() => this.tasks().filter(t => !t.isDone).length);
  readonly hasActive = computed(() => this.activeCount() > 0);
  /** 当前在弹窗中查看的任务 ID（空表示不显示进度弹窗） */
  readonly viewing = signal<string>('');
  private timers = new Map<string, ReturnType<typeof setTimeout>>();
  private polling = new Set<string>();
  private unknownCounts = new Map<string, number>();

  constructor() { this.tasks().forEach(t => { if (!t.isDone) this.startPolling(t.taskID); }); }

  /** 登记一个异步任务并开始轮询；refresh 表示完成后要联动刷新资源缓存 */
  track(taskID: string, title: string, refresh = false, resourceID = ''): void {
    if (!taskID) return;
    const now = Date.now();
    const existing = this.tasks().find(t => t.taskID === taskID);
    if (existing) { this.viewing.set(taskID); return; }
    const item: TaskItem = { taskID, title, percentage: 0, message: '任务已提交', detailMsg: '', isDone: false, failed: false, refresh, createdAt: now, updatedAt: now, resourceID };
    this.tasks.update(list => [item, ...list]);
    this.persist();
    this.viewing.set(taskID);
    this.startPolling(taskID);
  }

  /** 打开某个任务的进度弹窗 */
  view(taskID: string): void { this.viewing.set(taskID); }
  /** 关闭进度弹窗（任务继续在后台轮询） */
  closeView(): void { this.viewing.set(''); }

  /** 删除单个任务（停止其轮询） */
  remove(taskID: string): void {
    this.stopPolling(taskID);
    this.tasks.update(list => list.filter(t => t.taskID !== taskID));
    if (this.viewing() === taskID) this.viewing.set('');
    this.persist();
  }

  /** 清空全部任务 */
  clearAll(): void {
    this.timers.forEach(t => clearTimeout(t));
    this.timers.clear();
    this.tasks.set([]);
    this.viewing.set('');
    this.persist();
  }

  /** 清除已完成的任务，保留进行中的 */
  clearDone(): void {
    this.tasks().filter(t => t.isDone).forEach(t => this.stopPolling(t.taskID));
    this.tasks.update(list => list.filter(t => !t.isDone));
    if (this.viewing() && !this.tasks().some(t => t.taskID === this.viewing())) this.viewing.set('');
    this.persist();
  }

  private startPolling(taskID: string): void {
    if (this.timers.has(taskID) || this.polling.has(taskID)) return;
    this.poll(taskID);
  }

  private schedulePolling(taskID: string): void {
    if (this.timers.has(taskID) || this.polling.has(taskID)) return;
    const item = this.tasks().find(t => t.taskID === taskID);
    if (!item || item.isDone) return;
    this.timers.set(taskID, setTimeout(() => {
      this.timers.delete(taskID);
      this.poll(taskID);
    }, POLL_INTERVAL));
  }

  private stopPolling(taskID: string): void {
    const timer = this.timers.get(taskID);
    if (timer) { clearTimeout(timer); this.timers.delete(taskID); }
    this.polling.delete(taskID);
    this.unknownCounts.delete(taskID);
  }

  private poll(taskID: string): void {
    if (this.polling.has(taskID)) return;
    this.polling.add(taskID);
    const finish = () => {
      this.polling.delete(taskID);
      this.schedulePolling(taskID);
    };
    this.http.get<ApiResponse<ProgressData>>('/api/progress/' + encodeURIComponent(taskID)).subscribe({
      next: r => {
        if (r.code !== 200 || !r.data) {
          this.markUnknown(taskID, r.msg);
        } else {
          this.unknownCounts.delete(taskID);
          this.apply(taskID, r.data);
        }
        finish();
      },
      error: () => finish(),
    });
  }

  private apply(taskID: string, d: ProgressData): void {
    let doneNow = false; let refreshNeeded = false; let failedNow = false; let failureTitle = ''; let failureMessage = '';
    this.tasks.update(list => list.map(t => {
      if (t.taskID !== taskID || t.isDone) return t;
      const failed = d.isDone && (d.percentage < 100 || /失败|错误|error|fail/i.test(d.message || ''));
      if (d.isDone) {
        doneNow = true;
        refreshNeeded = t.refresh;
        failedNow = failed;
        failureTitle = d.name || t.title;
        failureMessage = d.detailMsg || d.message || '任务执行失败';
      }
      return { ...t, percentage: d.percentage, message: d.message || t.message, detailMsg: d.detailMsg || '', isDone: d.isDone, failed, updatedAt: Date.now() };
    }));
    if (doneNow) {
      this.stopPolling(taskID);
      if (failedNow) this.toast.error(failureTitle || '任务失败', failureMessage);
      if (refreshNeeded) this.bus.refresh(['containers', 'ports', 'images', 'compose']);
      this.completed.next({ taskID, resourceID: this.tasks().find(t => t.taskID === taskID)?.resourceID, refresh: refreshNeeded });
    }
    this.persist();
  }

  private markUnknown(taskID: string, msg: string): void {
    const attempts = (this.unknownCounts.get(taskID) || 0) + 1;
    this.unknownCounts.set(taskID, attempts);
    if (attempts < 4) return;
    let changed = false;
    this.tasks.update(list => list.map(t => {
      if (t.taskID !== taskID || t.isDone) return t;
      changed = true;
      return { ...t, isDone: true, failed: true, message: msg || '任务不存在或已过期', detailMsg: msg || '', updatedAt: Date.now() };
    }));
    this.stopPolling(taskID);
    if (changed) {
      this.toast.error('任务失败', msg || '任务不存在或已过期');
      this.completed.next({ taskID, resourceID: this.tasks().find(t => t.taskID === taskID)?.resourceID, refresh: false });
    }
    this.persist();
  }

  private persist(): void {
    try {
      const now = Date.now();
      const keep = this.tasks().filter(t => !t.isDone || now - t.updatedAt < DONE_KEEP);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(keep));
    } catch { /* localStorage 不可用则忽略 */ }
  }

  private restore(): TaskItem[] {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      const list = JSON.parse(raw) as TaskItem[];
      return Array.isArray(list) ? list : [];
    } catch { return []; }
  }
}
