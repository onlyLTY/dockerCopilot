import { Injectable, signal } from '@angular/core';

export type ToastKind = 'success' | 'error' | 'info';
export interface Toast { id: number; kind: ToastKind; title: string; message: string; }

/**
 * 全局轻提示服务：在页面居中上部弹出成功/失败/信息提示，默认数秒后自动关闭。
 */
@Injectable({ providedIn: 'root' })
export class ToastService {
  readonly toasts = signal<Toast[]>([]);
  private seq = 0;

  private push(kind: ToastKind, title: string, message: string, timeout: number): void {
    const id = ++this.seq;
    this.toasts.update(list => [...list, { id, kind, title, message }]);
    if (timeout > 0) setTimeout(() => this.dismiss(id), timeout);
  }

  success(message: string, timeout?: number): void;
  success(title: string, message: string, timeout?: number): void;
  success(titleOrMessage: string, messageOrTimeout?: string | number, timeout = 2000): void {
    this.show('success', '成功', titleOrMessage, messageOrTimeout, timeout);
  }

  error(message: string, timeout?: number): void;
  error(title: string, message: string, timeout?: number): void;
  error(titleOrMessage: string, messageOrTimeout?: string | number, timeout = 4000): void {
    this.show('error', '错误', titleOrMessage, messageOrTimeout, timeout);
  }

  info(message: string, timeout?: number): void;
  info(title: string, message: string, timeout?: number): void;
  info(titleOrMessage: string, messageOrTimeout?: string | number, timeout = 2500): void {
    this.show('info', '提示', titleOrMessage, messageOrTimeout, timeout);
  }

  dismiss(id: number): void { this.toasts.update(list => list.filter(t => t.id !== id)); }

  private show(kind: ToastKind, fallbackTitle: string, titleOrMessage: string, messageOrTimeout: string | number | undefined, timeout: number): void {
    if (typeof messageOrTimeout === 'string') {
      this.push(kind, titleOrMessage, messageOrTimeout, timeout);
      return;
    }
    this.push(kind, fallbackTitle, titleOrMessage, typeof messageOrTimeout === 'number' ? messageOrTimeout : timeout);
  }
}
