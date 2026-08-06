import { Component, HostListener, computed, inject } from '@angular/core';
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
export class TaskProgressComponent {
  readonly tasks = inject(TaskService);
  readonly task = computed(() => this.tasks.tasks().find(t => t.taskID === this.tasks.viewing()));

  @HostListener('document:keydown.escape') onEscape(): void {
    if (this.task()) this.tasks.closeView();
  }

  duration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(ms < 10000 ? 1 : 0)}s`;
  }

  close(e: Event) {
    if (e.target === e.currentTarget) this.tasks.closeView();
  }
  remove(taskID: string) {
    this.tasks.remove(taskID);
  }
}
