import { Component, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SettingsService } from '../../../../core/settings.service';
import { ToastService } from '../../../../core/toast.service';
import { FormExpansionComponent } from '../../../../shared/form-expansion/form-expansion.component';
import { FormSelectComponent, FormSelectOption } from '../../../../shared/form-select/form-select.component';
import { ModalHeadingComponent } from '../../../../shared/modal-heading/modal-heading.component';

@Component({
  selector: 'dc-settings-dialog',
  standalone: true,
  imports: [FormsModule, ModalHeadingComponent, FormSelectComponent, FormExpansionComponent],
  templateUrl: './settings-dialog.component.html',
  styleUrl: './settings-dialog.component.scss',
})
export class SettingsDialogComponent {
  private readonly settings = inject(SettingsService);
  private readonly toast = inject(ToastService);
  readonly closed = output<void>();
  readonly saving = signal(false);
  readonly settingsLoading = signal(true);
  readonly settingsLoadFailed = signal(false);
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

  updateDraft = '';
  backupDraft = '';
  logDraft = 'info';
  retentionDraft = 10;
  pullTimeoutDraft = 0;
  hubUrlsDraft: string[] = [];
  proxyDraft = { githubProxy: '' };

  private defaultHubUrls: string[] = [
    'docker.1ms.run', 'docker.m.daocloud.io', 'docker.1panel.top', 'docker.1panel.live',
    'proxy.1panel.live', 'dockerproxy.1panel.live', 'docker.1panel.dev', 'docker.anye.in',
    'hub.rat.dev', 'docker.amingg.com',
  ];
  private readonly updateLabels: Record<string, string> = {
    off: '关闭（仅手动检查）', '30m': '每 30 分钟', '1h': '每小时', '6h': '每 6 小时',
    '12h': '每 12 小时', '24h': '每天一次',
  };
  private readonly backupLabels: Record<string, string> = {
    off: '关闭（仅手动备份）', '6h': '每 6 小时', '12h': '每 12 小时',
    '24h': '每天一次', week: '每周', month: '每月',
  };

  constructor() {
    this.loadSettings();
  }

  updateLabel(key: string): string { return this.updateLabels[key] || key; }
  backupLabel(key: string): string { return this.backupLabels[key] || key; }

  reloadSettings(): void { this.loadSettings(); }
  close(): void { if (!this.saving()) this.closed.emit(); }

  private loadSettings(): void {
    this.settingsLoading.set(true);
    this.settingsLoadFailed.set(false);
    this.settings.refresh().subscribe({
      next: response => {
        if (response.code !== 200 || !response.data) {
          this.settingsLoading.set(false);
          this.settingsLoadFailed.set(true);
          return;
        }
        const data = response.data;
        this.settingsLoading.set(false);
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
      error: () => {
        this.settingsLoading.set(false);
        this.settingsLoadFailed.set(true);
        this.toast.error('设置加载失败，请重试');
      },
    });
  }

  addHubUrl(): void {
    if (this.hubUrlsDraft.length >= 20) { this.toast.error('最多 20 个加速源'); return; }
    this.hubUrlsDraft = [...this.hubUrlsDraft, ''];
  }
  removeHubUrl(index: number): void { this.hubUrlsDraft = this.hubUrlsDraft.filter((_, i) => i !== index); }
  updateHubUrl(index: number, value: string): void { const next = [...this.hubUrlsDraft]; next[index] = value; this.hubUrlsDraft = next; }
  resetHubUrls(): void { this.hubUrlsDraft = [...this.defaultHubUrls]; }
  trackHubUrl(index: number): number { return index; }

  saveSettings(): void {
    if (this.settingsLoading() || this.settingsLoadFailed()) { this.toast.error('设置尚未成功加载，暂时不能保存'); return; }
    const value = Number(this.retentionDraft);
    if (!Number.isInteger(value) || value < 1 || value > 100) { this.toast.error('保留数量必须是 1-100 的整数'); return; }
    const pullTimeout = Number(this.pullTimeoutDraft);
    if (!Number.isInteger(pullTimeout) || pullTimeout < 0 || pullTimeout > 3600) { this.toast.error('拉取镜像超时必须在 0-3600 秒之间'); return; }
    this.saving.set(true);
    this.settings.saveAll({
      updateCheckInterval: this.updateDraft, autoBackupInterval: this.backupDraft, logLevel: this.logDraft,
      retention: value, pullTimeoutSec: pullTimeout, hubUrls: this.hubUrlsDraft.map(x => x.trim()).filter(Boolean),
      proxy: this.proxyDraft,
    }).subscribe({
      next: response => {
        if (response.code === 200 && response.data) {
          this.pullTimeoutDraft = response.data.pullTimeoutSec ?? pullTimeout;
          if (response.data.hubUrls) this.hubUrlsDraft = [...response.data.hubUrls];
        }
        this.finishSave(value, response.code !== 200, response.msg);
      },
      error: () => this.finishSave(value, true),
    });
  }

  private finishSave(value: number, failed: boolean, msg?: string): void {
    this.saving.set(false);
    if (failed) { this.toast.error(msg || '设置保存失败'); return; }
    const runtime = this.settings.runtime();
    this.settings.setRuntime({ updateInterval: this.updateDraft, backupInterval: this.backupDraft, logLevel: this.logDraft, retention: value });
    this.closed.emit();
    const scope = [
      this.updateDraft !== runtime.updateInterval ? '更新检查频率已生效' : '',
      this.backupDraft !== runtime.backupInterval ? '自动备份频率已生效' : '',
      this.logDraft !== runtime.logLevel ? '日志级别已生效' : '',
      value !== runtime.retention ? '备份保留数量已生效' : '',
    ].filter(Boolean).join('；');
    this.toast.success(msg || `设置已保存${scope ? '：' + scope : ''}`);
  }
}
