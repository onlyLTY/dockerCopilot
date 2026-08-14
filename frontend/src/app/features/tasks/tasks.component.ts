import { Component, inject, computed, signal } from '@angular/core';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { ResourceCardComponent } from '../../shared/resource-card/resource-card.component';
import { MatTooltipModule } from '@angular/material/tooltip';

/**
 * 任务页：查看容器更新 / 备份恢复 / 部署等异步任务的列表与进度，
 * 支持打开进度弹窗、删除单个任务、清除已完成或全部任务。
 */
@Component({
  selector: 'dc-tasks',
  standalone: true,
  imports: [
    PageStateComponent,
    IconComponent,
    StatsComponent,
    PageHeadingComponent,
    ResourceCardComponent,
    MatTooltipModule,
  ],
  templateUrl: './tasks.component.html',
})
export class TasksComponent {
  readonly tasks = inject(TaskService);
  readonly confirm = inject(ConfirmService);
  readonly list = this.tasks.tasks;
  readonly activeCount = this.tasks.activeCount;
  readonly doneCount = computed(() => this.list().filter(t => t.isDone && !t.failed).length);
  readonly failedCount = computed(() => this.list().filter(t => t.failed).length);
  readonly filter = signal('all');
  readonly filteredTasks = computed(() =>
    this.list().filter(
      t =>
        this.filter() === 'all' ||
        (this.filter() === 'active'
          ? !t.isDone
          : this.filter() === 'done'
            ? t.isDone && !t.failed
            : t.failed),
    ),
  );
  readonly stats = computed<readonly StatItem[]>(() => [
    { key: 'all', value: this.list().length, label: '总任务' },
    { key: 'active', value: this.activeCount(), label: '进行中', tone: 'amber' },
    { key: 'done', value: this.doneCount(), label: '已完成', tone: 'green' },
    { key: 'failed', value: this.failedCount(), label: '失败', tone: 'red' },
  ]);

  selectFilter(key: string): void {
    this.filter.set(this.filter() === key || key === 'all' ? 'all' : key);
  }
  view(taskID: string) {
    this.tasks.view(taskID);
  }
  remove(taskID: string) {
    this.tasks.remove(taskID);
  }
  clearDone() {
    this.tasks.clearDone();
  }
  async stopAll() {
    const active = this.activeCount();
    if (!active) return;
    if (await this.confirm.open({
      title: '停止全部任务',
      message: `将请求停止全部 ${active} 个进行中任务。已执行的 Docker 操作不会自动回滚，确定继续吗？`,
      confirmText: '停止全部',
      danger: true,
    })) {
      const count = await this.tasks.cancelAllActive();
      if (count) this.tasks.viewing.set('');
    }
  }

  cancel(taskID: string) {
    return this.tasks.cancel(taskID);
  }
  statusText(t: { isDone: boolean; failed: boolean; canceled?: boolean; timedOut?: boolean }) {
    if (t.canceled) return '已取消';
    if (t.timedOut) return '已超时';
    return t.failed ? '失败' : t.isDone ? '已完成' : '进行中';
  }
  time(ts: number) {
    return new Date(ts).toLocaleString();
  }
}
