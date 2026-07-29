import { Component, HostListener, computed, inject } from '@angular/core';
import { Router } from '@angular/router';
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
        <section class="modal modal-content modal-m" role="dialog" aria-modal="true" (click)="$event.stopPropagation()">
          <div class="modal-head">
            <div>
              <p class="eyebrow">TASK PROGRESS</p>
              <h2>{{ t.title }}</h2>
              <p>{{ t.message }}</p>
            </div>
            <button class="modal-close" (click)="tasks.closeView()" aria-label="关闭">×</button>
          </div>

          <div class="modal-body task-content">
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
            <div class="task-actions d-flex items-center justify-end gap-8">
              <button class="danger-link d-inline-flex items-center gap-4" (click)="remove(t.taskID)"><dc-icon class="icon-sm" name="delete"></dc-icon>删除任务</button>
              <button class="secondary d-inline-flex items-center gap-4" (click)="goTasks()"><dc-icon class="icon-sm" name="tasks"></dc-icon>前往任务</button>
              <button class="primary d-inline-flex items-center justify-center" (click)="tasks.closeView()">关闭</button>
            </div>
          </div>
        </section>
      </div>
    }
  `,
})
export class TaskProgressComponent {
  readonly tasks = inject(TaskService);
  readonly router = inject(Router);
  readonly task = computed(() => this.tasks.tasks().find(t => t.taskID === this.tasks.viewing()));

  @HostListener('document:keydown.escape') onEscape(): void {
    if (this.task()) this.tasks.closeView();
  }

  goTasks() { this.tasks.closeView(); this.router.navigateByUrl('/tasks'); }
  close(e: Event) { if (e.target === e.currentTarget) this.tasks.closeView(); }
  remove(taskID: string) { this.tasks.remove(taskID); }
}
