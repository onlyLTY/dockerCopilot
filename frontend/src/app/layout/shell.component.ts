import { Component, effect, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs/operators';
import { AuthService } from '../core/auth.service';
import { VersionService } from '../core/version.service';
import { UiStateService } from '../core/ui-state.service';
import { TaskService } from '../core/task.service';
import { IconComponent } from '../shared/icon/icon.component';

@Component({
  selector: 'dc-shell',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './shell.component.html',
  styleUrl: './shell.component.scss',
})
export class ShellComponent {
  readonly auth = inject(AuthService);
  readonly ui = inject(UiStateService);
  readonly tasks = inject(TaskService);
  private readonly router = inject(Router);
  private readonly versionService = inject(VersionService);
  open = false;
  readonly version = signal({ version: '', buildDate: '' });
  /** 当前路由 path，用 signal 驱动无 zone 下菜单选中态刷新 */
  private readonly currentPath = signal(this.normalizePath(this.router.url));
  constructor() {
    this.router.events
      .pipe(
        filter((e): e is NavigationEnd => e instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(e => {
        this.currentPath.set(this.normalizePath(e.urlAfterRedirects || e.url));
      });
    effect(() => {
      if (!this.auth.authenticated()) {
        this.version.set({ version: '', buildDate: '' });
        return;
      }
      this.versionService.local().subscribe({
        next: result => {
          if (result.code === 200) this.version.set(result.data);
        },
        error: () => this.version.set({ version: '', buildDate: '' }),
      });
    });
  }
  navigate(path: string): void {
    this.router.navigateByUrl(path);
    this.closeMenu();
  }
  isActive(path: string): boolean {
    const current = this.currentPath();
    const target = path.replace(/^\//, '');
    return current === target || current.startsWith(target + '/');
  }
  closeMenu(): void {
    this.open = false;
  }
  logout(): void {
    this.auth.logout();
    this.closeMenu();
    this.router.navigateByUrl('/login');
  }
  private normalizePath(url: string): string {
    return url.split('?')[0].split('#')[0].replace(/^\//, '');
  }
}
