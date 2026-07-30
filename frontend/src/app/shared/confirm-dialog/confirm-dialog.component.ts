import { Component, HostListener, inject } from '@angular/core';
import { ConfirmService } from '../../core/confirm.service';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'dc-confirm-dialog',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './confirm-dialog.component.html',
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
