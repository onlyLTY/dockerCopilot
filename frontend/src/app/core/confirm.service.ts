import { Injectable, signal } from '@angular/core';

export interface ConfirmOptions {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  critical?: boolean;
}

interface PendingConfirm {
  options: ConfirmOptions;
  resolve: (value: boolean) => void;
}

@Injectable({ providedIn: 'root' })
export class ConfirmService {
  readonly dialog = signal<PendingConfirm | undefined>(undefined);

  open(options: ConfirmOptions): Promise<boolean> {
    const current = this.dialog();
    if (current) current.resolve(false);
    return new Promise(resolve => this.dialog.set({ options, resolve }));
  }

  close(result: boolean): void {
    const current = this.dialog();
    if (!current) return;
    current.resolve(result);
    this.dialog.set(undefined);
  }
}
