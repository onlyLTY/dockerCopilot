import { Component, inject } from '@angular/core';
import { ToastService } from '../../core/toast.service';

/**
 * 轻提示宿主：渲染在页面居中上部，成功/失败/信息各有样式，点击可手动关闭。
 */
@Component({
  selector: 'dc-toast',
  standalone: true,
  templateUrl: './toast.component.html',
  styleUrl: './toast.component.scss',
})
export class ToastComponent {
  readonly toast = inject(ToastService);
}
