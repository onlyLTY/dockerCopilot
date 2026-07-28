import { Component, inject, computed } from '@angular/core';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

/**
 * 任务页：查看容器更新 / 备份恢复 / 部署等异步任务的列表与进度，
 * 支持打开进度弹窗、删除单个任务、清除已完成或全部任务。
 */
@Component({
  selector: 'dc-tasks',
  standalone: true,
  imports: [PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './tasks.component.html',
})
export class TasksComponent {
  readonly tasks = inject(TaskService);
  readonly list = this.tasks.tasks;
  readonly activeCount = this.tasks.activeCount;
  readonly doneCount = computed(() => this.list().filter(t => t.isDone && !t.failed).length);
  readonly failedCount = computed(() => this.list().filter(t => t.failed).length);

  view(taskID: string) { this.tasks.view(taskID); }
  remove(taskID: string) { this.tasks.remove(taskID); }
  clearDone() { this.tasks.clearDone(); }
  clearAll() { if (confirm('清空全部任务记录？')) this.tasks.clearAll(); }
  statusText(t: { isDone: boolean; failed: boolean }) { return t.failed ? '失败' : t.isDone ? '已完成' : '进行中'; }
  time(ts: number) { return new Date(ts).toLocaleString(); }
}
