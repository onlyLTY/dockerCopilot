import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthService } from '../core/auth.service';
import { VersionService } from '../core/version.service';
import { UiStateService } from '../core/ui-state.service';
import { IconComponent } from '../shared/icon.component';

@Component({
  selector: 'dc-shell',
  standalone: true,
  imports: [CommonModule, IconComponent],
  templateUrl: './shell.component.html',
})
export class ShellComponent {
  readonly auth = inject(AuthService); readonly ui = inject(UiStateService); private readonly router = inject(Router); private readonly versionService = inject(VersionService);
  open = false; version = { version: '', buildDate: '' };
  constructor() { if (this.auth.isAuthenticated()) this.versionService.local().subscribe({ next: result => { if (result.code === 200) this.version = result.data; } }); }
  navigate(path: string): void { this.router.navigateByUrl(path); this.closeMenu(); }
  isActive(path: string): boolean { return this.router.url.startsWith(path); }
  closeMenu(): void { this.open = false; }
  logout(): void { this.auth.logout(); this.closeMenu(); this.router.navigateByUrl('/login'); }
}
