import { Component, computed, inject } from '@angular/core';
import { TaskService } from '../core/task.service';
import { IconComponent } from './icon.component';

/**
 * 任务进度弹窗：展示 TaskService.viewing 指向的任务进度。
 * 关闭仅隐藏弹窗，任务继续在后台轮询；也可在此从任务列表中删除该任务。
 */
@Component({
  selector: 'dc-task-progress',
  standalone: true,
  imports: [IconComponent],
  template: `
    @if (task(); as t) {
      <div class="modal-backdrop" (click)="close($event)">
        <section class="modal task-modal" role="dialog" aria-modal="true">
          <div class="modal-head">
            <div>
              <p class="eyebrow">TASK PROGRESS</p>
              <h2>{{ t.title }}</h2>
              <p>{{ t.message }}</p>
            </div>
            <button class="modal-close" (click)="tasks.closeView()" aria-label="关闭">×</button>
          </div>

          <div class="task-progress">
            <div class="progress-bar" [class.failed]="t.failed" [class.done]="t.isDone && !t.failed">
              <span [style.width.%]="t.percentage"></span>
            </div>
            <div class="progress-meta">
              <span [class.failed]="t.failed">
                {{ t.failed ? '失败' : t.isDone ? '已完成' : '进行中' }} · {{ t.percentage }}%
              </span>
              @if (!t.isDone) { <span class="muted">后台执行中，可关闭此窗口</span> }
            </div>
            @if (t.detailMsg) {
              <pre class="task-detail">{{ t.detailMsg }}</pre>
            }
          </div>

          <div class="modal-actions">
            <button class="danger-link" (click)="remove(t.taskID)"><dc-icon name="delete"></dc-icon>删除任务</button>
            <button class="primary" (click)="tasks.closeView()">关闭</button>
          </div>
        </section>
      </div>
    }
  `,
})
export class TaskProgressComponent {
  readonly tasks = inject(TaskService);
  readonly task = computed(() => this.tasks.tasks().find(t => t.taskID === this.tasks.viewing()));
  close(e: Event) { if (e.target === e.currentTarget) this.tasks.closeView(); }
  remove(taskID: string) { this.tasks.remove(taskID); }
}
