import { Component, HostListener, computed, inject } from '@angular/core';
import { Router } from '@angular/router';
import { TaskService } from '../../core/task.service';
import { IconComponent } from '../icon/icon.component';

/**
 * 任务进度弹窗：展示 TaskService.viewing 指向的任务进度。
 * 关闭仅隐藏弹窗，任务继续在后台轮询；也可在此从任务列表中删除该任务。
 */
@Component({
  selector: 'dc-task-progress',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './task-progress.component.html',
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
