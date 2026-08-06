import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SettingsService } from '../../core/settings.service';
import { VersionService, VersionInfo } from '../../core/version.service';
import { ToastService } from '../../core/toast.service';
import { ApiResponse } from '../../core/compose.service';
import { HttpClient } from '@angular/common/http';
import { IconComponent } from '../../shared/icon/icon.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { ModalHeadingComponent } from '../../shared/modal-heading/modal-heading.component';
import {
  FormSelectComponent,
  FormSelectOption,
} from '../../shared/form-select/form-select.component';
import { FormExpansionComponent } from '../../shared/form-expansion/form-expansion.component';

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

@Component({
  selector: 'dc-me',
  standalone: true,
  imports: [
    FormsModule,
    IconComponent,
    StatsComponent,
    PageHeadingComponent,
    ModalHeadingComponent,
    FormSelectComponent,
    FormExpansionComponent,
  ],
  templateUrl: './me.component.html',
  styleUrl: './me.component.scss',
})
export class MeComponent {
  private readonly settings = inject(SettingsService);
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
  // 日志弹窗：按等级筛选；切换时重新请求后端（level=error 为最近 N 条 error，非混合 100 条里筛）
  readonly logLevelFilter = signal<string>('error');
  /** 当前结果条数（服务端已按 level 过滤） */
  readonly logResultCount = computed(() => this.logs().length);
  readonly filteredLogs = computed<LogEntry[]>(() => this.logs());
  // 等级 tab：当前选中项显示本次拉到的条数；其它项为 0（点击后会重新请求）
  readonly logLevelStats = computed<StatItem[]>(() => {
    const n = this.logResultCount();
    const f = this.logLevelFilter();
    const v = (key: string) => (f === key ? n : 0);
    return [
      { key: 'error', value: v('error'), label: 'ERROR', tone: 'red' },
      { key: 'warn', value: v('warn'), label: 'WARN', tone: 'amber' },
      { key: 'info', value: v('info'), label: 'INFO', tone: 'blue' },
      { key: 'debug', value: v('debug'), label: 'DEBUG' },
    ];
  });
  readonly updateOptions = signal<string[]>([]);
  readonly backupOptions = signal<string[]>([]);
  readonly logOptions = signal<string[]>(['debug', 'info', 'warn', 'error']);
  readonly updateSelectOptions = computed<FormSelectOption[]>(() =>
    this.updateOptions().map(value => ({ value, label: this.updateLabel(value) })),
  );
  readonly backupSelectOptions = computed<FormSelectOption[]>(() =>
    this.backupOptions().map(value => ({ value, label: this.backupLabel(value) })),
  );
  readonly logSelectOptions = computed<FormSelectOption[]>(() =>
    this.logOptions().map(value => ({ value, label: value })),
  );
  readonly stats = computed<readonly StatItem[]>(() => [
    { value: this.updateLabel(this.runtime().updateInterval), label: '更新检查', tone: 'blue' },
    { value: this.backupLabel(this.runtime().backupInterval), label: '自动备份', tone: 'green' },
    { value: this.runtime().retention, label: '备份保留', tone: 'violet' },
    { value: this.runtime().logLevel, label: '日志级别', tone: 'amber' },
  ]);
  get updateInterval() {
    return this.runtime().updateInterval;
  }
  get backupInterval() {
    return this.runtime().backupInterval;
  }
  get logLevel() {
    return this.runtime().logLevel;
  }
  get retention() {
    return this.runtime().retention;
  }
  updateDraft = '';
  backupDraft = '';
  logDraft = 'info';
  retentionDraft = 10;
  /** 拉取镜像超时（秒），0 表示未配置 */
  pullTimeoutDraft = 0;
  /** Docker Hub 加速源（有序）。空字符串表示正在编辑的新行，保存时会被过滤。 */
  hubUrlsDraft: string[] = [];
  /** 后端下发的默认列表；请求失败时用本地兜底。 */
  private defaultHubUrls: string[] = [
    'docker.1ms.run',
    'docker.m.daocloud.io',
    'docker.1panel.top',
    'docker.1panel.live',
    'proxy.1panel.live',
    'dockerproxy.1panel.live',
    'docker.1panel.dev',
    'docker.anye.in',
    'hub.rat.dev',
    'docker.amingg.com',
  ];
  proxyDraft = { githubProxy: '', HTTP_PROXY: '', HTTPS_PROXY: '', NO_PROXY: '' };

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
      next: r => {
        if (r.code === 200) this.version.set(r.data);
      },
      error: () => this.version.set({ version: '', buildDate: '' }),
    });
    this.loadSettings();
  }

  updateLabel(key: string) {
    return this.updateLabels[key] || key;
  }
  backupLabel(key: string) {
    return this.backupLabels[key] || key;
  }
  openSettings() {
    this.loadSettings();
    this.showSettings.set(true);
  }
  closeSettings() {
    if (!this.saving()) this.showSettings.set(false);
  }

  private loadSettings() {
    this.settings.getAll().subscribe({
      next: r => {
        if (r.code !== 200 || !r.data) return;
        const data = r.data;
        this.settings.setRuntime({
          updateInterval: data.updateCheck?.interval ?? '',
          backupInterval: data.autoBackup?.interval ?? '',
          logLevel: data.logLevel?.level ?? 'info',
          retention: data.retention ?? 10,
        });
        this.updateOptions.set(data.updateCheck?.options || []);
        this.backupOptions.set(data.autoBackup?.options || []);
        this.logOptions.set(data.logLevel?.options || this.logOptions());
        this.updateDraft = data.updateCheck?.interval || '';
        this.backupDraft = data.autoBackup?.interval || '';
        this.logDraft = data.logLevel?.level || 'info';
        this.retentionDraft = data.retention ?? 10;
        this.pullTimeoutDraft = data.pullTimeoutSec ?? 0;
        if (data.defaultHubUrls?.length) this.defaultHubUrls = [...data.defaultHubUrls];
        this.hubUrlsDraft = [...(data.hubUrls ?? this.defaultHubUrls)];
        if (data.proxy) this.proxyDraft = { ...data.proxy };
      },
    });
  }

  addHubUrl() {
    if (this.hubUrlsDraft.length >= 20) {
      this.toast.error('最多 20 个加速源');
      return;
    }
    this.hubUrlsDraft = [...this.hubUrlsDraft, ''];
  }

  removeHubUrl(index: number) {
    this.hubUrlsDraft = this.hubUrlsDraft.filter((_, i) => i !== index);
  }

  updateHubUrl(index: number, value: string) {
    const next = [...this.hubUrlsDraft];
    next[index] = value;
    this.hubUrlsDraft = next;
  }

  resetHubUrls() {
    this.hubUrlsDraft = [...this.defaultHubUrls];
  }

  trackHubUrl(index: number): number {
    return index;
  }

  saveSettings() {
    const value = Number(this.retentionDraft);
    if (!Number.isInteger(value) || value < 1 || value > 100) {
      this.toast.error('保留数量必须是 1-100 的整数');
      return;
    }
    const pullTimeout = Number(this.pullTimeoutDraft);
    if (!Number.isInteger(pullTimeout) || pullTimeout < 0 || pullTimeout > 3600) {
      this.toast.error('拉取镜像超时必须在 0-3600 秒之间，0 表示未配置');
      return;
    }
    const hubUrls = this.hubUrlsDraft.map(x => x.trim()).filter(Boolean);
    this.saving.set(true);
    this.settings
      .saveAll({
        updateCheckInterval: this.updateDraft,
        autoBackupInterval: this.backupDraft,
        logLevel: this.logDraft,
        retention: value,
        pullTimeoutSec: pullTimeout,
        hubUrls,
        proxy: this.proxyDraft,
      })
      .subscribe({
        next: r => {
          if (r.code === 200 && r.data?.hubUrls) this.hubUrlsDraft = [...r.data.hubUrls];
          this.finishSave(value, r.code !== 200, r.msg);
        },
        error: () => this.finishSave(value, true),
      });
  }

  private finishSave(value: number, failed: boolean, msg?: string) {
    this.saving.set(false);
    if (failed) {
      this.toast.error(msg || '设置保存失败');
      return;
    }
    this.settings.setRuntime({
      updateInterval: this.updateDraft,
      backupInterval: this.backupDraft,
      logLevel: this.logDraft,
      retention: value,
    });
    this.showSettings.set(false);
    this.toast.success(msg || '设置已保存，代理设置将在重启服务后生效');
  }

  openLogs() {
    this.showLogs.set(true);
    this.logLevelFilter.set('error');
    this.fetchLogs('error');
  }
  closeLogs() {
    this.showLogs.set(false);
    this.logLevelFilter.set('error');
    this.logs.set([]);
  }
  selectLogLevel(key: string) {
    if (!key || key === this.logLevelFilter()) return;
    this.logLevelFilter.set(key);
    this.fetchLogs(key);
  }

  private fetchLogs(level: string) {
    this.loadingLogs.set(true);
    this.logError.set('');
    const params = new URLSearchParams({ limit: '100', level });
    this.http
      .get<ApiResponse<{ entries: LogEntry[] }>>('/api/logs?' + params.toString())
      .subscribe({
        next: r => {
          this.loadingLogs.set(false);
          if (r.code === 200) this.logs.set(r.data?.entries || []);
          else this.logError.set(r.msg || '读取日志失败');
        },
        error: e => {
          this.loadingLogs.set(false);
          this.logError.set(e.error?.msg || '读取日志失败');
        },
      });
  }
}
