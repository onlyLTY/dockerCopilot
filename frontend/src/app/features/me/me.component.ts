import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { SettingsService } from '../../core/settings.service';
import { BackupService } from '../../core/backup.service';
import { VersionService, VersionInfo } from '../../core/version.service';
import { ToastService } from '../../core/toast.service';
import { ApiResponse } from '../../core/compose.service';
import { HttpClient } from '@angular/common/http';
import { IconComponent } from '../../shared/icon/icon.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { ModalHeadingComponent } from '../../shared/modal-heading/modal-heading.component';
import { FormSelectComponent, FormSelectOption } from '../../shared/form-select/form-select.component';
import { FormExpansionComponent } from '../../shared/form-expansion/form-expansion.component';

interface LogEntry { timestamp: string; level: string; message: string; }
interface LogLevelData { level: string; options: string[]; }

@Component({
  selector: 'dc-me',
  standalone: true,
  imports: [FormsModule, IconComponent, StatsComponent, PageHeadingComponent, ModalHeadingComponent, FormSelectComponent, FormExpansionComponent],
  templateUrl: './me.component.html',
})
export class MeComponent {
  private readonly settings = inject(SettingsService);
  private readonly backups = inject(BackupService);
  private readonly versions = inject(VersionService);
  private readonly http = inject(HttpClient);
  private readonly toast = inject(ToastService);

  readonly runtime = this.settings.runtime;
  readonly version = signal<VersionInfo>({ version: '', buildDate: '' });
  readonly showSettings = signal(false);
  readonly showLogs = signal(false);
  readonly saving = signal(false);
  readonly loadingLogs = signal(false);
  readonly logError = signal('');
  readonly logs = signal<LogEntry[]>([]);
  // 日志弹窗：按等级筛选的 tab，'all' = 全部
  readonly logLevelFilter = signal<string>('all');
  readonly logLevels = signal<string[]>(['debug', 'info', 'warn', 'error']);
  readonly logLevelCount = computed<Record<string, number>>(() => {
    const counts: Record<string, number> = { all: this.logs().length };
    for (const e of this.logs()) counts[e.level] = (counts[e.level] || 0) + 1;
    return counts;
  });
  readonly filteredLogs = computed<LogEntry[]>(() => {
    const f = this.logLevelFilter();
    return f === 'all' ? this.logs() : this.logs().filter(e => e.level === f);
  });
  // 等级筛选 stats：全部(无 tone) + 各等级带 tone
  readonly logLevelStats = computed<StatItem[]>(() => {
    const counts = this.logLevelCount();
    return [
      { key: 'all', value: counts['all'] || 0, label: '全部' },
      { key: 'debug', value: counts['debug'] || 0, label: 'DEBUG' },
      { key: 'info', value: counts['info'] || 0, label: 'INFO', tone: 'blue' },
      { key: 'warn', value: counts['warn'] || 0, label: 'WARN', tone: 'amber' },
      { key: 'error', value: counts['error'] || 0, label: 'ERROR', tone: 'red' },
    ];
  });
  readonly updateOptions = signal<string[]>([]);
  readonly backupOptions = signal<string[]>([]);
  readonly logOptions = signal<string[]>(['debug', 'info', 'warn', 'error']);
  readonly updateSelectOptions = computed<FormSelectOption[]>(() => this.updateOptions().map(value => ({ value, label: this.updateLabel(value) })));
  readonly backupSelectOptions = computed<FormSelectOption[]>(() => this.backupOptions().map(value => ({ value, label: this.backupLabel(value) })));
  readonly logSelectOptions = computed<FormSelectOption[]>(() => this.logOptions().map(value => ({ value, label: value })));
  readonly stats = computed<readonly StatItem[]>(() => [
    { value: this.updateLabel(this.runtime().updateInterval), label: '更新检查', tone: 'blue' },
    { value: this.backupLabel(this.runtime().backupInterval), label: '自动备份', tone: 'green' },
    { value: this.runtime().retention, label: '备份保留', tone: 'violet' },
    { value: this.runtime().logLevel, label: '日志级别', tone: 'amber' },
  ]);
  get updateInterval() { return this.runtime().updateInterval; }
  get backupInterval() { return this.runtime().backupInterval; }
  get logLevel() { return this.runtime().logLevel; }
  get retention() { return this.runtime().retention; }
  updateDraft = ''; backupDraft = ''; logDraft = 'info'; retentionDraft = 10;
  proxyDraft = { githubProxy: '', HTTP_PROXY: '', HTTPS_PROXY: '', NO_PROXY: '' };

  private readonly updateLabels: Record<string, string> = { off: '关闭（仅手动检查）', '30m': '每 30 分钟', '1h': '每小时', '6h': '每 6 小时', '12h': '每 12 小时', '24h': '每天一次' };
  private readonly backupLabels: Record<string, string> = { off: '关闭（仅手动备份）', '6h': '每 6 小时', '12h': '每 12 小时', '24h': '每天一次', week: '每周', month: '每月' };

  constructor() {
    this.versions.local().subscribe({
      next: r => { if (r.code === 200) this.version.set(r.data); },
      error: () => this.version.set({ version: '', buildDate: '' }),
    });
    this.loadSettings();
  }

  updateLabel(key: string) { return this.updateLabels[key] || key; }
  backupLabel(key: string) { return this.backupLabels[key] || key; }
  openSettings() { this.loadSettings(); this.showSettings.set(true); }
  closeSettings() { if (!this.saving()) this.showSettings.set(false); }

  private loadSettings() {
    this.settings.load();
    this.settings.getUpdateSettings().subscribe({ next: r => { if (r.code === 200) { this.updateOptions.set(r.data.options || []); this.updateDraft = this.runtime().updateInterval || r.data.interval; } } });
    this.settings.getBackupSettings().subscribe({ next: r => { if (r.code === 200) { this.backupOptions.set(r.data.options || []); this.backupDraft = this.runtime().backupInterval || r.data.interval; } } });
    this.settings.getLogSettings().subscribe({ next: r => { if (r.code === 200) { this.logOptions.set(r.data.options || this.logOptions()); this.logDraft = this.runtime().logLevel || r.data.level; } } });
    this.backups.settings().subscribe({ next: r => { if (r.code === 200) { this.retentionDraft = r.data.retention; this.settings.setRuntime({ retention: r.data.retention }); } } });
    this.settings.getProxySettings().subscribe({ next: r => { if (r.code === 200 && r.data) this.proxyDraft = { ...r.data }; } });
  }

  saveSettings() {
    const value = Number(this.retentionDraft);
    if (!Number.isInteger(value) || value < 1 || value > 100) { this.toast.error('保留数量必须是 1-100 的整数'); return; }
    this.saving.set(true);
    forkJoin([
      this.settings.setUpdateInterval(this.updateDraft),
      this.settings.setBackupInterval(this.backupDraft),
      this.settings.setLogLevel(this.logDraft),
      this.backups.updateSettings(value),
      this.settings.setProxySettings(this.proxyDraft),
    ]).subscribe({ next: results => this.finishSave(value, results.some(r => r.code !== 200)), error: () => this.finishSave(value, true) });
  }

  private finishSave(value: number, failed: boolean) {
    this.saving.set(false);
    if (failed) { this.toast.error('部分设置保存失败'); return; }
    this.settings.setRuntime({ updateInterval: this.updateDraft, backupInterval: this.backupDraft, logLevel: this.logDraft, retention: value });
    this.showSettings.set(false); this.toast.success('设置已保存，代理设置将在重启服务后生效');
  }

  openLogs() {
    this.showLogs.set(true); this.loadingLogs.set(true); this.logError.set('');
    this.http.get<ApiResponse<{ entries: LogEntry[] }>>('/api/logs?limit=100').subscribe({ next: r => { this.loadingLogs.set(false); if (r.code === 200) this.logs.set(r.data?.entries || []); else this.logError.set(r.msg || '读取日志失败'); }, error: e => { this.loadingLogs.set(false); this.logError.set(e.error?.msg || '读取日志失败'); } });
  }
  closeLogs() { this.showLogs.set(false); this.logLevelFilter.set('all'); }
  // dc-stats 在再次点击当前 active 项时 emit ''，此时归一回「全部」
  selectLogLevel(key: string) { this.logLevelFilter.set(key || 'all'); }
}
