import { Component, HostListener, inject } from '@angular/core';
import { ConfirmService } from '../core/confirm.service';

@Component({
  selector: 'dc-confirm-dialog',
  standalone: true,
  template: `
    @if (confirm.dialog(); as dialog) {
      <div class="modal-backdrop confirm-backdrop" (click)="closeBackdrop($event)">
        <section class="modal modal-confirm modal-m" role="dialog" aria-modal="true" (click)="$event.stopPropagation()">
          <div class="modal-head">
            <div><h2>{{ dialog.options.title }}</h2></div>
            <button class="modal-close" (click)="confirm.close(false)" aria-label="关闭">×</button>
          </div>
          <div class="modal-body">
            <p class="confirm-message">{{ dialog.options.message }}</p>
          </div>
          <div class="modal-actions d-flex items-center justify-end gap-8">
            <button class="secondary d-inline-flex items-center justify-center" (click)="confirm.close(false)">{{ dialog.options.cancelText || '取消' }}</button>
            <button class="d-inline-flex items-center justify-center" [class.primary]="!dialog.options.danger" [class.danger-button]="dialog.options.danger" (click)="confirm.close(true)">{{ dialog.options.confirmText || '确定' }}</button>
          </div>
        </section>
      </div>
    }
  `,
})
export class ConfirmDialogComponent {
  readonly confirm = inject(ConfirmService);

  @HostListener('document:keydown.escape') onEscape(): void {
    if (this.confirm.dialog()) this.confirm.close(false);
  }

  closeBackdrop(event: Event): void {
    if (event.target === event.currentTarget) this.confirm.close(false);
  }
}
