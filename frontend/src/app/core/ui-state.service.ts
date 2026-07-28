import { Injectable, signal } from '@angular/core';

export type ThemeMode = 'light' | 'dark';

@Injectable({ providedIn: 'root' })
export class UiStateService {
  readonly theme = signal<ThemeMode>((localStorage.getItem('dc-theme') as ThemeMode) || 'light');
  readonly compact = signal(localStorage.getItem('dc-compact') === 'true');

  constructor() { this.applyTheme(); }
  toggleTheme(): void { const next = this.theme() === 'light' ? 'dark' : 'light'; this.theme.set(next); localStorage.setItem('dc-theme', next); this.applyTheme(); }
  toggleCompact(): void { const next = !this.compact(); this.compact.set(next); localStorage.setItem('dc-compact', String(next)); }
  private applyTheme(): void { document.documentElement.dataset['theme'] = this.theme(); }
}
