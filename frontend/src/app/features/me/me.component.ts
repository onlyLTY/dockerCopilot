import { Component, computed, inject, signal } from '@angular/core';
import { SettingsService } from '../../core/settings.service';
import { VersionService, VersionInfo } from '../../core/version.service';
import { IconComponent } from '../../shared/icon/icon.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { SettingsDialogComponent } from './components/settings-dialog/settings-dialog.component';
import { LogDialogComponent } from './components/log-dialog/log-dialog.component';

@Component({
  selector: 'dc-me',
  standalone: true,
  imports: [
    IconComponent,
    StatsComponent,
    PageHeadingComponent,
    SettingsDialogComponent,
    LogDialogComponent,
  ],
  templateUrl: './me.component.html',
  styleUrl: './me.component.scss',
})
export class MeComponent {
  private readonly settings = inject(SettingsService);
  private readonly versions = inject(VersionService);

  readonly runtime = this.settings.runtime;
  readonly version = signal<VersionInfo>({ version: '', buildDate: '' });
  readonly showSettings = signal(false);
  readonly showLogs = signal(false);
  readonly stats = computed<readonly StatItem[]>(() => [
    {
      value: this.updateLabel(this.runtime().updateInterval),
      label: '更新检查',
      tone: 'blue',
    },
    {
      value: this.backupLabel(this.runtime().backupInterval),
      label: '自动备份',
      tone: 'green',
    },
    { value: this.runtime().retention, label: '备份保留', tone: 'violet' },
    { value: this.runtime().logLevel, label: '日志级别', tone: 'amber' },
  ]);

  private readonly updateLabels: Record<string, string> = {
    off: '关闭（仅手动检查）',
    '30m': '每 30 分钟',
    '1h': '每小时',
    '6h': '每 6 小时',
    '12h': '每 12 小时',
    '24h': '每天一次',
  };
  private readonly backupLabels: Record<string, string> = {
    off: '关闭（仅手动备份）',
    '6h': '每 6 小时',
    '12h': '每 12 小时',
    '24h': '每天一次',
    week: '每周',
    month: '每月',
  };

  constructor() {
    this.versions.local().subscribe({
      next: response => {
        if (response.code === 200) this.version.set(response.data);
      },
      error: () => this.version.set({ version: '', buildDate: '' }),
    });
  }

  updateLabel(key: string): string {
    return this.updateLabels[key] || key;
  }

  backupLabel(key: string): string {
    return this.backupLabels[key] || key;
  }

  openSettings(): void {
    this.showSettings.set(true);
  }

  closeSettings(): void {
    this.showSettings.set(false);
  }

  openLogs(): void {
    this.showLogs.set(true);
  }

  closeLogs(): void {
    this.showLogs.set(false);
  }
}
