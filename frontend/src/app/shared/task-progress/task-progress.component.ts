import { Component, HostListener, OnDestroy, computed, inject, signal } from '@angular/core';
import { TaskService, TaskItem } from '../../core/task.service';
import { IconComponent } from '../icon/icon.component';
import { ModalHeadingComponent } from '../modal-heading/modal-heading.component';

/**
 * 任务进度弹窗：展示 TaskService.viewing 指向的任务进度。
 * 关闭仅隐藏弹窗，任务继续在后台轮询；也可在此从任务列表中删除该任务。
 */
@Component({
  selector: 'dc-task-progress',
  standalone: true,
  imports: [IconComponent, ModalHeadingComponent],
  templateUrl: './task-progress.component.html',
  styleUrl: './task-progress.component.scss',
})
export class TaskProgressComponent implements OnDestroy {
  readonly tasks = inject(TaskService);
  readonly task = computed(() => this.tasks.tasks().find(t => t.taskID === this.tasks.viewing()));
  readonly now = signal(Date.now());
  private readonly timer = setInterval(() => this.now.set(Date.now()), 1000);

  @HostListener('document:keydown.escape') onEscape(): void {
    if (this.task()) this.tasks.closeView();
  }

  ngOnDestroy(): void {
    clearInterval(this.timer);
  }

  duration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)}s`;
  }

  stepDuration(step: { startedAt: number; endedAt?: number; durationMs?: number }): string {
    const elapsed = step.endedAt
      ? (step.durationMs ?? Math.max(0, step.endedAt - step.startedAt))
      : Math.max(0, this.now() - step.startedAt);
    return this.duration(elapsed);
  }

  taskDuration(task: TaskItem): string {
    if (task.isDone && typeof task.durationMs === 'number' && task.durationMs >= 0) {
      return this.duration(task.durationMs);
    }
    const startedAt = task.startedAt || (task.steps || [])[0]?.startedAt;
    if (startedAt) {
      const endedAt = task.isDone
        ? task.endedAt || task.updatedAt
        : this.now();
      return this.duration(Math.max(0, endedAt - startedAt));
    }
    const steps = task.steps || [];
    if (steps.length) {
      const lastStep = steps[steps.length - 1];
      const endedAt = task.isDone
        ? lastStep.endedAt || task.updatedAt
        : this.now();
      return this.duration(Math.max(0, endedAt - steps[0].startedAt));
    }
    const endedAt = task.isDone ? task.updatedAt : this.now();
    return this.duration(Math.max(0, endedAt - task.createdAt));
  }

  formatBytes(bytes: number): string {
    if (bytes < 0) bytes = 0;
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let unit = 0;
    let value = bytes;
    while (value >= 1024 && unit < units.length - 1) {
      value /= 1024;
      unit++;
    }
    return unit === 0 ? `${Math.round(value)} ${units[unit]}` : `${value.toFixed(1)} ${units[unit]}`;
  }

  close(e: Event) {
    if (e.target === e.currentTarget) this.tasks.closeView();
  }
  remove(taskID: string) {
    this.tasks.remove(taskID);
  }
}
