import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from '../core/auth.service';
import { VersionService } from '../core/version.service';
import { UiStateService } from '../core/ui-state.service';
import { TaskService } from '../core/task.service';
import { IconComponent } from '../shared/icon.component';

@Component({
  selector: 'dc-shell',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './shell.component.html',
})
export class ShellComponent {
  readonly auth = inject(AuthService); readonly ui = inject(UiStateService); readonly tasks = inject(TaskService); private readonly router = inject(Router); private readonly versionService = inject(VersionService);
  open = false; readonly version = signal({ version: '', buildDate: '' });
  constructor() { if (this.auth.isAuthenticated()) this.versionService.local().subscribe({ next: result => { if (result.code === 200) this.version.set(result.data); } }); }
  navigate(path: string): void { this.router.navigateByUrl(path); this.closeMenu(); }
  isActive(path: string): boolean { return this.router.url.startsWith(path); }
  closeMenu(): void { this.open = false; }
  logout(): void { this.auth.logout(); this.closeMenu(); this.router.navigateByUrl('/login'); }
}
