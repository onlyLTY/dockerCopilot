import { Injectable, signal } from '@angular/core';

export type ToastKind = 'success' | 'error' | 'info';
export interface Toast { id: number; kind: ToastKind; text: string; }

/**
 * 全局轻提示服务：在页面居中上部弹出成功/失败/信息提示，默认数秒后自动关闭。
 */
@Injectable({ providedIn: 'root' })
export class ToastService {
  readonly toasts = signal<Toast[]>([]);
  private seq = 0;

  private push(kind: ToastKind, text: string, timeout: number): void {
    const id = ++this.seq;
    this.toasts.update(list => [...list, { id, kind, text }]);
    if (timeout > 0) setTimeout(() => this.dismiss(id), timeout);
  }

  success(text: string, timeout = 2000): void { this.push('success', text, timeout); }
  error(text: string, timeout = 4000): void { this.push('error', text, timeout); }
  info(text: string, timeout = 2500): void { this.push('info', text, timeout); }

  dismiss(id: number): void { this.toasts.update(list => list.filter(t => t.id !== id)); }
}
