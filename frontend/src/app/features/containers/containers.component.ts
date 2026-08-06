import { Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { from, forkJoin, of } from 'rxjs';
import { catchError, map, mergeMap, toArray } from 'rxjs/operators';
import { ContainerService, ContainerRow } from '../../core/container.service';
import { IconService } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { ConfirmService } from '../../core/confirm.service';
import { SoftRefreshHandle, startSoftRefresh } from '../../core/soft-refresh';
import { actionErrorMessage, actionLabel, runAction } from '../../core/run-action';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ResourceCardComponent } from '../../shared/resource-card/resource-card.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';

@Component({
  selector: 'dc-containers',
  standalone: true,
  imports: [
    PageStateComponent,
    IconComponent,
    ResourceCardComponent,
    StatsComponent,
    PageHeadingComponent,
  ],
  templateUrl: './containers.component.html',
})
export class ContainersComponent {
  private readonly service = inject(ContainerService);
  private readonly icons = inject(IconService);
  private readonly toast = inject(ToastService);
  private readonly tasks = inject(TaskService);
  private readonly confirm = inject(ConfirmService);
  private readonly destroyRef = inject(DestroyRef);
  private softRefresh: SoftRefreshHandle | null = null;
  readonly containers = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly selected = signal<Set<string>>(new Set());
  readonly selectionMode = signal(false);
  readonly busy = signal(false);
  /** 批量操作进行中的动作标签（启动/停止/更新），用于工具栏按钮文案 */
  readonly bulkAction = signal<string | null>(null);
  readonly checking = signal(false);
  readonly activeUpdateIds = signal<Set<string>>(new Set());
  readonly updateIgnoreBusyIds = signal<Set<string>>(new Set());
  /** 单容器启停/重启进行中，按 id 记录当前动作标签，用于禁用按钮并展示「启动中」等文案 */
  readonly actionBusy = signal<Map<string, string>>(new Map());
  readonly filter = signal('all');
  readonly runningCount = computed(() => this.containers().filter(x => this.isRunning(x)).length);
  readonly updateCount = computed(() => this.containers().filter(x => x.haveUpdate).length);
  readonly ignoredUpdateCount = computed(
    () => this.containers().filter(x => x.updateIgnored).length,
  );
  readonly filteredContainers = computed(() =>
    this.containers().filter(x => {
      switch (this.filter()) {
        case 'running':
          return this.isRunning(x);
        case 'stopped':
          return !this.isRunning(x);
        case 'update':
          return x.haveUpdate;
        case 'ignored':
          return x.updateIgnored;
        default:
          return true;
      }
    }),
  );
  readonly stats = computed<readonly StatItem[]>(() => [
    { key: 'all', value: this.containers().length, label: '总容器' },
    { key: 'running', value: this.runningCount(), label: '运行中', tone: 'green' },
    {
      key: 'stopped',
      value: this.containers().length - this.runningCount(),
      label: '已停止',
      tone: 'red',
    },
    { key: 'update', value: this.updateCount(), label: '有更新', tone: 'amber' },
    { key: 'ignored', value: this.ignoredUpdateCount(), label: '已忽略更新', tone: 'violet' },
  ]);
  readonly selectedCount = computed(() => this.selected().size);
  readonly allSelected = computed(
    () =>
      this.filteredContainers().length > 0 &&
      this.filteredContainers().every(x => this.selected().has(x.id)),
  );
  readonly hasActiveUpdates = computed(() => this.activeUpdateIds().size > 0);
  readonly availableUpdateCount = computed(
    () => this.containers().filter(x => x.haveUpdate && !this.activeUpdateIds().has(x.id)).length,
  );
  constructor() {
    this.service.ensureLoaded();
    this.icons.ensureLoaded();
    this.tasks.completed.pipe(takeUntilDestroyed()).subscribe(({ resourceID }) => {
      if (!resourceID) return;
      const next = new Set(this.activeUpdateIds());
      next.delete(resourceID);
      this.activeUpdateIds.set(next);
    });
    this.softRefresh = startSoftRefresh({
      intervalMs: 20_000,
      refresh: () => this.service.refresh(),
      shouldSkip: () => this.busy() || this.checking() || this.loading(),
    });
    this.destroyRef.onDestroy(() => this.softRefresh?.stop());
  }
  refresh(): void {
    this.service.refresh();
  }
  selectFilter(key: string): void {
    this.filter.set(this.filter() === key || key === 'all' ? 'all' : key);
    this.selected.set(new Set());
  }
  // 手动检查更新：异步任务，登记进度；完成后刷新容器列表以更新“有更新”标识
  checkUpdate(): void {
    if (this.checking()) return;
    runAction({
      request: this.service.checkUpdate(),
      onStart: () => this.checking.set(true),
      onFinally: () => this.checking.set(false),
      onSuccess: r => {
        const taskID = (r.data as { taskID?: string } | undefined)?.taskID;
        if (taskID) this.tasks.track(String(taskID), '检查更新', true);
        else this.toast.info('检查更新任务已提交');
      },
      onBizError: r => this.toast.error(`检查更新失败：${r.msg || '未知错误'}`),
      onHttpError: e => this.toast.error(`检查更新失败：${actionErrorMessage(e)}`),
    });
  }
  hasUpdate(x: ContainerRow) {
    return !!x.haveUpdate;
  }
  isRunning(x: ContainerRow) {
    return x.status.includes('Up') || x.status.includes('running') || x.status.includes('运行');
  }
  icon(x: ContainerRow) {
    return this.icons.resolve(x.usingImage, this.iconMap());
  }
  fallback(e: Event) {
    (e.target as HTMLImageElement).src = this.icons.defaultIcon();
  }

  // 将 Docker 状态文本（如 "Up 3 hours"）解析为本地化时长
  private duration(text: string): string {
    if (!text) return '';
    if (/less than a second/i.test(text)) return '刚刚';
    let s = text
      .replace(/^Up\s+/i, '')
      .replace(/\s+\(.*\)/, '')
      .replace(/^About\s+/i, '')
      .trim();
    const m = s.match(/^(\d+|an?)\s+(second|minute|hour|day|week|month|year)s?/i);
    if (!m) return '';
    const num = m[1] === 'a' || m[1] === 'an' ? 1 : parseInt(m[1], 10);
    const unit: Record<string, string> = {
      second: '秒',
      minute: '分钟',
      hour: '小时',
      day: '天',
      week: '周',
      month: '个月',
      year: '年',
    };
    return `${num}${unit[m[2].toLowerCase()]}`;
  }
  private stateLabel(state: string): string {
    const map: Record<string, string> = {
      running: '运行中',
      exited: '已停止',
      created: '已创建',
      restarting: '重启中',
      paused: '已暂停',
      dead: '已停止',
      removing: '删除中',
    };
    return map[(state || '').toLowerCase()] || state || '未知';
  }
  // 卡片上的一行：运行中显示 “运行：30分钟”，其余显示本地化状态
  statusText(x: ContainerRow): string {
    if (this.isRunning(x)) {
      const d = this.duration(x.runningTime);
      return d ? `运行：${d}` : '运行中';
    }
    return this.stateLabel(x.status);
  }

  enterSelection() {
    this.selectionMode.set(true);
  }
  exitSelection() {
    this.selectionMode.set(false);
    this.selected.set(new Set());
  }
  toggleSelection() {
    this.selectionMode() ? this.exitSelection() : this.enterSelection();
  }

  isSelected(id: string) {
    return this.selected().has(id);
  }
  toggleSelect(id: string) {
    const next = new Set(this.selected());
    next.has(id) ? next.delete(id) : next.add(id);
    this.selected.set(next);
  }
  toggleAll() {
    const visible = this.filteredContainers();
    const current = this.selected();
    const allVisible = visible.length > 0 && visible.every(x => current.has(x.id));
    this.selected.set(
      allVisible
        ? new Set([...current].filter(id => !visible.some(x => x.id === id)))
        : new Set([...current, ...visible.map(x => x.id)]),
    );
  }

  isActionBusy(id: string): boolean {
    return this.actionBusy().has(id);
  }
  actionBusyLabel(id: string): string | undefined {
    return this.actionBusy().get(id);
  }
  private run(x: ContainerRow, fn: (id: string) => any, label: string, async = false) {
    if (this.actionBusy().has(x.id)) return;
    runAction({
      request: fn(x.id),
      onStart: () => this.actionBusy.update(m => new Map(m).set(x.id, label)),
      onFinally: () => this.clearActionBusy(x.id),
      onSuccess: () => {
        if (async) this.toast.info(`${x.name} ${label}任务已提交`);
        else this.toast.success(actionLabel(x.name, label, true));
      },
      onBizError: r => this.toast.error(actionLabel(x.name, label, false, r.msg || '未知错误')),
      onHttpError: e => this.toast.error(actionLabel(x.name, label, false, actionErrorMessage(e))),
    });
  }
  start(x: ContainerRow) {
    this.run(x, id => this.service.start(id), '启动');
  }
  stop(x: ContainerRow) {
    this.run(x, id => this.service.stop(id), '停止');
  }
  restart(x: ContainerRow) {
    this.run(x, id => this.service.restart(id), '重启');
  }
  async remove(x: ContainerRow) {
    if (this.actionBusy().has(x.id)) return;
    const running = this.isRunning(x);
    const label = running ? '强制删除' : '删除';
    if (
      !(await this.confirm.open({
        title: `${label}容器`,
        message: running
          ? `容器 ${x.name} 正在运行，确定强制停止并删除吗？此操作不可恢复。`
          : `确定删除容器 ${x.name} 吗？此操作不可恢复。`,
        confirmText: label,
        danger: true,
        critical: running,
      }))
    )
      return;
    this.run(x, id => this.service.remove(id, running), label);
  }
  private clearActionBusy(id: string): void {
    this.actionBusy.update(m => {
      const next = new Map(m);
      next.delete(id);
      return next;
    });
  }
  // 更新是异步任务：拿到 taskID 后登记进度，弹窗展示进度
  update(x: ContainerRow) {
    if (this.activeUpdateIds().has(x.id)) return;
    runAction({
      request: this.service.update(x.id, x.usingImage, x.name),
      onStart: () => this.markUpdateActive(x.id),
      onSuccess: (r: any) => {
        const taskID = r.data?.taskID;
        if (taskID) this.tasks.track(String(taskID), '更新 ' + x.name, true, x.id);
        else this.toast.info(`${x.name} 更新任务已提交`);
      },
      onBizError: r => {
        this.markUpdateInactive(x.id);
        this.toast.error(actionLabel(x.name, '更新', false, r.msg || '未知错误'));
      },
      onHttpError: e => {
        this.markUpdateInactive(x.id);
        this.toast.error(actionLabel(x.name, '更新', false, actionErrorMessage(e)));
      },
    });
  }

  ignoreUpdate(x: ContainerRow) {
    this.setUpdateIgnored(x, true);
  }
  restoreUpdate(x: ContainerRow) {
    this.setUpdateIgnored(x, false);
  }
  private setUpdateIgnored(x: ContainerRow, ignored: boolean): void {
    if (this.updateIgnoreBusyIds().has(x.id)) return;
    const label = ignored ? '忽略更新' : '恢复检测';
    const request = ignored ? this.service.ignoreUpdate(x.id) : this.service.restoreUpdate(x.id);
    runAction({
      request,
      onStart: () => this.updateIgnoreBusyIds.update(ids => new Set(ids).add(x.id)),
      onFinally: () =>
        this.updateIgnoreBusyIds.update(ids => {
          const copy = new Set(ids);
          copy.delete(x.id);
          return copy;
        }),
      onBizError: r => this.toast.error(actionLabel(x.name, label, false, r.msg || '未知错误')),
      onHttpError: e => this.toast.error(actionLabel(x.name, label, false, actionErrorMessage(e))),
    });
  }
  private bulk(fn: (id: string) => any, label: string, async = false) {
    const targets = this.filteredContainers().filter(x => this.selected().has(x.id));
    if (!targets.length || this.busy()) return;
    this.busy.set(true);
    this.bulkAction.set(label);
    // 批量期间锁定各卡片按钮，并给出即时反馈
    this.actionBusy.update(m => {
      const next = new Map(m);
      targets.forEach(x => next.set(x.id, label));
      return next;
    });
    forkJoin(
      targets.map(x =>
        fn(x.id).pipe(
          map((r: any) => ({ name: x.name, ok: r.code === 200, msg: r.msg })),
          catchError((e: any) =>
            of({ name: x.name, ok: false, msg: e.error?.msg || e.message || '请求错误' }),
          ),
        ),
      ),
    ).subscribe(results => {
      this.busy.set(false);
      this.bulkAction.set(null);
      this.actionBusy.update(m => {
        const next = new Map(m);
        targets.forEach(x => next.delete(x.id));
        return next;
      });
      const ok = results.filter(r => r.ok).length;
      const fail = results.length - ok;
      if (fail === 0)
        async
          ? this.toast.info(`已提交 ${ok} 个容器的${label}任务`)
          : this.toast.success(`已${label} ${ok} 个容器`);
      else {
        const first = results.find(r => !r.ok);
        this.toast.error(`${label}完成 ${ok} 个，失败 ${fail} 个${first ? '：' + first.msg : ''}`);
      }
      this.selected.set(new Set());
    });
  }
  bulkStart() {
    this.bulk(id => this.service.start(id), '启动');
  }
  bulkStop() {
    this.bulk(id => this.service.stop(id), '停止');
  }
  async bulkRemove() {
    const targets = this.filteredContainers().filter(x => this.selected().has(x.id));
    if (!targets.length || this.busy()) return;
    const runningCount = targets.filter(x => this.isRunning(x)).length;
    if (
      !(await this.confirm.open({
        title: '批量删除容器',
        message:
          runningCount > 0
            ? `将删除 ${targets.length} 个容器（其中 ${runningCount} 个正在运行，会先强制停止）。此操作不可恢复，确定继续吗？`
            : `将删除 ${targets.length} 个容器。此操作不可恢复，确定继续吗？`,
        confirmText: '确认删除',
        danger: true,
        critical: runningCount > 0,
      }))
    )
      return;
    // 运行中的容器用 force，已停止的普通删除
    const targetsById = new Map(targets.map(x => [x.id, x]));
    this.bulk(id => {
      const item = targetsById.get(id);
      return this.service.remove(id, item ? this.isRunning(item) : true);
    }, '删除');
  }
  // 批量更新使用固定并发窗口，避免一次性启动大量 Docker 重建任务
  bulkUpdate() {
    this.submitUpdates(
      this.filteredContainers().filter(
        x => this.selected().has(x.id) && !this.activeUpdateIds().has(x.id),
      ),
    );
  }
  async updateAll() {
    const targets = this.containers().filter(
      x => x.haveUpdate && !this.activeUpdateIds().has(x.id),
    );
    if (!targets.length || this.busy()) {
      if (!targets.length) this.toast.info('当前没有可更新的容器');
      return;
    }
    if (
      !(await this.confirm.open({
        title: '一键更新容器',
        message: `将更新 ${targets.length} 个容器，并在更新过程中短暂停止服务。确定继续吗？`,
        confirmText: '确认更新',
      }))
    )
      return;
    this.submitUpdates(targets);
  }
  private submitUpdates(targets: ContainerRow[]): void {
    if (!targets.length || this.busy()) return;
    this.busy.set(true);
    this.bulkAction.set('更新');
    targets.forEach(x => this.markUpdateActive(x.id));
    from(targets)
      .pipe(
        mergeMap(
          x =>
            this.service.update(x.id, x.usingImage, x.name).pipe(
              map((r: any) => ({
                container: x,
                ok: r.code === 200,
                taskID: r.data?.taskID,
                msg: r.msg,
              })),
              catchError((e: any) =>
                of({
                  container: x,
                  ok: false,
                  taskID: undefined,
                  msg: e.error?.msg || e.message || '请求错误',
                }),
              ),
            ),
          3,
        ),
        toArray(),
      )
      .subscribe(results => {
        this.busy.set(false);
        this.bulkAction.set(null);
        results.forEach(r => {
          if (r.ok && r.taskID)
            this.tasks.track(String(r.taskID), '更新 ' + r.container.name, true, r.container.id);
          else this.markUpdateInactive(r.container.id);
        });
        const ok = results.filter(r => r.ok).length;
        const fail = results.length - ok;
        if (fail === 0) this.toast.info(`已提交 ${ok} 个容器的更新任务`);
        else {
          const first = results.find(r => !r.ok);
          this.toast.error(`更新提交 ${ok} 个，失败 ${fail} 个${first ? '：' + first.msg : ''}`);
        }
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
