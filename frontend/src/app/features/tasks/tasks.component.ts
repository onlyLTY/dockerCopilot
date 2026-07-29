import { Component, inject, computed } from '@angular/core';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';

/**
 * 任务页：查看容器更新 / 备份恢复 / 部署等异步任务的列表与进度，
 * 支持打开进度弹窗、删除单个任务、清除已完成或全部任务。
 */
@Component({
  selector: 'dc-tasks',
  standalone: true,
  imports: [PageStateComponent, IconComponent],
  templateUrl: './tasks.component.html',
})
export class TasksComponent {
  readonly tasks = inject(TaskService); readonly confirm = inject(ConfirmService);
  readonly list = this.tasks.tasks;
  readonly activeCount = this.tasks.activeCount;
  readonly doneCount = computed(() => this.list().filter(t => t.isDone && !t.failed).length);
  readonly failedCount = computed(() => this.list().filter(t => t.failed).length);

  view(taskID: string) { this.tasks.view(taskID); }
  remove(taskID: string) { this.tasks.remove(taskID); }
  clearDone() { this.tasks.clearDone(); }
  async clearAll() { if (await this.confirm.open({ title: '清空任务记录', message: '确定清空全部任务记录吗？', confirmText: '清空全部', danger: true })) this.tasks.clearAll(); }
  statusText(t: { isDone: boolean; failed: boolean }) { return t.failed ? '失败' : t.isDone ? '已完成' : '进行中'; }
  time(ts: number) { return new Date(ts).toLocaleString(); }
}
