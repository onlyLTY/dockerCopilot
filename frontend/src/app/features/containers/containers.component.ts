import { Component, computed, inject, signal } from '@angular/core';
import { from, forkJoin, of } from 'rxjs';
import { catchError, map, mergeMap, toArray } from 'rxjs/operators';
import { ContainerService, ContainerRow } from '../../core/container.service';
import { IconService } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-containers',
  standalone: true,
  imports: [PageStateComponent, IconComponent],
  templateUrl: './containers.component.html',
})
export class ContainersComponent {
  private readonly service = inject(ContainerService); private readonly icons = inject(IconService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService);
  // 数据、加载态、错误态均来自服务里的常驻缓存，页面切换不再重复请求
  readonly containers = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly selected = signal<Set<string>>(new Set());
  readonly selectionMode = signal(false);
  readonly busy = signal(false);
  readonly checking = signal(false);
  readonly activeUpdateIds = signal<Set<string>>(new Set());
  readonly runningCount = computed(() => this.containers().filter(x => this.isRunning(x)).length); readonly updateCount = computed(() => this.containers().filter(x => x.haveUpdate).length);
  readonly selectedCount = computed(() => this.selected().size);
  readonly allSelected = computed(() => this.containers().length > 0 && this.selected().size === this.containers().length);
  readonly hasActiveUpdates = computed(() => this.activeUpdateIds().size > 0);
  constructor() {
    this.service.ensureLoaded();
    this.icons.ensureLoaded();
    this.tasks.completed.subscribe(({ resourceID }) => {
      if (!resourceID) return;
      const next = new Set(this.activeUpdateIds());
      next.delete(resourceID);
      this.activeUpdateIds.set(next);
    });
  }
  refresh(): void { this.service.refresh(); }
  // 手动检查更新：异步任务，登记进度；完成后刷新容器列表以更新“有更新”标识
  checkUpdate(): void {
    if (this.checking()) return;
    this.checking.set(true);
    this.service.checkUpdate().subscribe({
      next: r => {
        this.checking.set(false);
        const taskID = r.data?.taskID;
        if (r.code === 200 && taskID) this.tasks.track(String(taskID), '检查更新', true);
        else if (r.code === 200) this.toast.info('检查更新任务已提交');
        else this.toast.error(`检查更新失败：${r.msg || '未知错误'}`);
      },
      error: e => { this.checking.set(false); this.toast.error(`检查更新失败：${e.error?.msg || e.message || '请求错误'}`); },
    });
  }
  hasUpdate(x: ContainerRow) { return !!x.haveUpdate; }
  isRunning(x: ContainerRow) { return x.status.includes('Up') || x.status.includes('running') || x.status.includes('运行'); }
  icon(x: ContainerRow) { return this.icons.resolve(x.usingImage, this.iconMap()); } fallback(e: Event) { (e.target as HTMLImageElement).src = this.icons.actionIcon('containers'); }

  // 将 Docker 状态文本（如 "Up 3 hours"）解析为本地化时长
  private duration(text: string): string {
    if (!text) return '';
    if (/less than a second/i.test(text)) return '刚刚';
    let s = text.replace(/^Up\s+/i, '').replace(/\s+\(.*\)/, '').replace(/^About\s+/i, '').trim();
    const m = s.match(/^(\d+|an?)\s+(second|minute|hour|day|week|month|year)s?/i);
    if (!m) return '';
    const num = (m[1] === 'a' || m[1] === 'an') ? 1 : parseInt(m[1], 10);
    const unit: Record<string, string> = { second: '秒', minute: '分钟', hour: '小时', day: '天', week: '周', month: '个月', year: '年' };
    return `${num}${unit[m[2].toLowerCase()]}`;
  }
  private stateLabel(state: string): string {
    const map: Record<string, string> = { running: '运行中', exited: '已停止', created: '已创建', restarting: '重启中', paused: '已暂停', dead: '已停止', removing: '删除中' };
    return map[(state || '').toLowerCase()] || state || '未知';
  }
  // 卡片上的一行：运行中显示 “运行：30分钟”，其余显示本地化状态
  statusText(x: ContainerRow): string {
    if (this.isRunning(x)) { const d = this.duration(x.runningTime); return d ? `运行：${d}` : '运行中'; }
    return this.stateLabel(x.status);
  }

  enterSelection() { this.selectionMode.set(true); }
  exitSelection() { this.selectionMode.set(false); this.selected.set(new Set()); }

  isSelected(id: string) { return this.selected().has(id); }
  toggleSelect(id: string) { const next = new Set(this.selected()); next.has(id) ? next.delete(id) : next.add(id); this.selected.set(next); }
  toggleAll() { this.selected.set(this.allSelected() ? new Set() : new Set(this.containers().map(x => x.id))); }

  // ===== 单个操作 =====
  private run(x: ContainerRow, fn: (id: string) => any, label: string, async = false) {
    fn(x.id).subscribe({
      next: (r: any) => {
        if (r.code === 200) { async ? this.toast.info(`${x.name} ${label}任务已提交`) : this.toast.success(`${x.name} ${label}成功`); }
        else this.toast.error(`${x.name} ${label}失败：${r.msg || '未知错误'}`);
      },
      error: (e: any) => this.toast.error(`${x.name} ${label}失败：${e.error?.msg || e.message || '请求错误'}`),
    });
  }
  start(x: ContainerRow) { this.run(x, id => this.service.start(id), '启动'); }
  stop(x: ContainerRow) { this.run(x, id => this.service.stop(id), '停止'); }
  restart(x: ContainerRow) { this.run(x, id => this.service.restart(id), '重启'); }
  // 更新是异步任务：拿到 taskID 后登记进度，弹窗展示进度
  update(x: ContainerRow) {
    if (this.activeUpdateIds().has(x.id)) return;
    this.markUpdateActive(x.id);
    this.service.update(x.id, x.usingImage, x.name).subscribe({
      next: (r: any) => {
        const taskID = r.data?.taskID;
        if (r.code === 200 && taskID) this.tasks.track(String(taskID), '更新 ' + x.name, true, x.id);
        else {
          this.markUpdateInactive(x.id);
          if (r.code === 200) this.toast.info(`${x.name} 更新任务已提交`);
          else this.toast.error(`${x.name} 更新失败：${r.msg || '未知错误'}`);
        }
      },
      error: (e: any) => {
        this.markUpdateInactive(x.id);
        this.toast.error(`${x.name} 更新失败：${e.error?.msg || e.message || '请求错误'}`);
      },
    });
  }

  // ===== 批量操作 =====
  private bulk(fn: (id: string) => any, label: string, async = false) {
    const targets = this.containers().filter(x => this.selected().has(x.id));
    if (!targets.length || this.busy()) return;
    this.busy.set(true);
    forkJoin(targets.map(x => fn(x.id).pipe(
      map((r: any) => ({ name: x.name, ok: r.code === 200, msg: r.msg })),
      catchError((e: any) => of({ name: x.name, ok: false, msg: e.error?.msg || e.message || '请求错误' })),
    ))).subscribe(results => {
      this.busy.set(false);
      const ok = results.filter(r => r.ok).length; const fail = results.length - ok;
      if (fail === 0) async ? this.toast.info(`已提交 ${ok} 个容器的${label}任务`) : this.toast.success(`已${label} ${ok} 个容器`);
      else { const first = results.find(r => !r.ok); this.toast.error(`${label}完成 ${ok} 个，失败 ${fail} 个${first ? '：' + first.msg : ''}`); }
      this.selected.set(new Set());
    });
  }
  bulkStart() { this.bulk(id => this.service.start(id), '启动'); }
  bulkStop() { this.bulk(id => this.service.stop(id), '停止'); }
  // 批量更新使用固定并发窗口，避免一次性启动大量 Docker 重建任务
  bulkUpdate() {
    const targets = this.containers().filter(x => this.selected().has(x.id) && !this.activeUpdateIds().has(x.id));
    if (!targets.length || this.busy()) return;
    this.busy.set(true);
    targets.forEach(x => this.markUpdateActive(x.id));
    from(targets).pipe(
      mergeMap(x => this.service.update(x.id, x.usingImage, x.name).pipe(
        map((r: any) => ({ container: x, ok: r.code === 200, taskID: r.data?.taskID, msg: r.msg })),
        catchError((e: any) => of({ container: x, ok: false, taskID: undefined, msg: e.error?.msg || e.message || '请求错误' })),
      ), 3),
      toArray(),
    ).subscribe(results => {
      this.busy.set(false);
      results.forEach(r => {
        if (r.ok && r.taskID) this.tasks.track(String(r.taskID), '更新 ' + r.container.name, true, r.container.id);
        else this.markUpdateInactive(r.container.id);
      });
      const ok = results.filter(r => r.ok).length; const fail = results.length - ok;
      if (fail === 0) this.toast.info(`已提交 ${ok} 个容器的更新任务`);
      else { const first = results.find(r => !r.ok); this.toast.error(`更新提交 ${ok} 个，失败 ${fail} 个${first ? '：' + first.msg : ''}`); }
      this.selected.set(new Set());
    });
  }

  private markUpdateActive(id: string): void {
    const next = new Set(this.activeUpdateIds());
    next.add(id);
    this.activeUpdateIds.set(next);
  }

  private markUpdateInactive(id: string): void {
    const next = new Set(this.activeUpdateIds());
    next.delete(id);
    this.activeUpdateIds.set(next);
  }
}
