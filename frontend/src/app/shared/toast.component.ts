import { Component, inject } from '@angular/core';
import { ToastService } from '../core/toast.service';

/**
 * 轻提示宿主：渲染在页面居中上部，成功/失败/信息各有样式，点击可手动关闭。
 */
@Component({
  selector: 'dc-toast',
  standalone: true,
  template: `
    <div class="toast-stack">
      @for (t of toast.toasts(); track t.id) {
        <button class="toast" [class.success]="t.kind === 'success'" [class.error]="t.kind === 'error'" [class.info]="t.kind === 'info'" (click)="toast.dismiss(t.id)">
          <span class="toast-mark">{{ t.kind === 'success' ? '✓' : t.kind === 'error' ? '✕' : 'ⓘ' }}</span>
          <span class="toast-text"><strong>{{ t.title }}</strong><span>{{ t.message }}</span></span>
        </button>
      }
    </div>
  `,
})
export class ToastComponent {
  readonly toast = inject(ToastService);
}
