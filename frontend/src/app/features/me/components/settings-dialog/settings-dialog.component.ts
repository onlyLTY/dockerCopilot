import { Component, DestroyRef, computed, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  DaemonProxySettings,
  DaemonRestartOperation,
  SettingsService,
} from '../../../../core/settings.service';
import { ToastService } from '../../../../core/toast.service';
import { ConfirmService } from '../../../../core/confirm.service';
import { FormExpansionComponent } from '../../../../shared/form-expansion/form-expansion.component';
import {
  FormSelectComponent,
  FormSelectOption,
} from '../../../../shared/form-select/form-select.component';
import { ModalHeadingComponent } from '../../../../shared/modal-heading/modal-heading.component';

@Component({
  selector: 'dc-settings-dialog',
  standalone: true,
  imports: [
    FormsModule,
    ModalHeadingComponent,
    FormSelectComponent,
    FormExpansionComponent,
  ],
  templateUrl: './settings-dialog.component.html',
  styleUrl: './settings-dialog.component.scss',
})
export class SettingsDialogComponent {
  private readonly settings = inject(SettingsService);
  private readonly toast = inject(ToastService);
  private readonly confirm = inject(ConfirmService);
  private readonly destroyRef = inject(DestroyRef);
  private daemonPollTimer: ReturnType<typeof setTimeout> | null = null;
  private daemonPollStartedAt = 0;
  private daemonPollFailures = 0;

  readonly closed = output<void>();
  readonly daemonProxy = signal<DaemonProxySettings | null>(null);
  readonly daemonProxyLoading = signal(false);
  readonly daemonProxyBusy = signal(false);
  readonly saving = signal(false);
  readonly daemonRestart = signal<DaemonRestartOperation | null>(null);
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
  proxyDraft = { githubProxy: '', HTTP_PROXY: '', HTTPS_PROXY: '', NO_PROXY: '' };
  daemonProxyDraft = { httpProxy: '', httpsProxy: '', noProxy: '' };
  daemonProxyErrors = { httpProxy: '', httpsProxy: '', noProxy: '' };

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
  private daemonProxyDraftConfigured = false;
  private daemonProxyHash = '';

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
    this.destroyRef.onDestroy(() => {
      if (this.daemonPollTimer) clearTimeout(this.daemonPollTimer);
    });
    this.loadSettings();
  }

  updateLabel(key: string): string {
    return this.updateLabels[key] || key;
  }

  backupLabel(key: string): string {
    return this.backupLabels[key] || key;
  }

  close(): void {
    if (!this.saving()) this.closed.emit();
  }

  private loadSettings(): void {
    this.settings.getAll().subscribe({
      next: response => {
        if (response.code !== 200 || !response.data) return;
        const data = response.data;
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
        if (data.daemonProxyDraftConfigured && data.daemonProxyDraft) {
          this.daemonProxyDraftConfigured = true;
          this.daemonProxyDraft = { ...this.daemonProxyDraft, ...data.daemonProxyDraft };
        }
        this.loadDaemonProxy();
      },
    });
  }

  private loadDaemonProxy(): void {
    this.daemonProxyLoading.set(true);
    this.settings.getDaemonProxy().subscribe({
      next: response => {
        this.daemonProxyLoading.set(false);
        if (response.code !== 200 || !response.data) {
          this.toast.error(response.msg || '读取 Docker daemon 代理失败');
          return;
        }
        const data = response.data;
        this.daemonProxy.set(data);
        this.daemonProxyHash = data.hash || '';
        if (!this.daemonProxyDraftConfigured) {
          const source = data.fileProxy || data.effectiveProxy;
          this.daemonProxyDraft = { ...this.daemonProxyDraft, ...source };
        }
      },
      error: error => {
        this.daemonProxyLoading.set(false);
        this.toast.error(error?.error?.msg || '读取 Docker daemon 代理失败');
      },
    });
  }

  private validateDaemonProxy(): boolean {
    const errors = { httpProxy: '', httpsProxy: '', noProxy: '' };
    for (const key of ['httpProxy', 'httpsProxy'] as const) {
      const value = this.daemonProxyDraft[key];
      if (!value) continue;
      if (/\s|[\u0000-\u001f\u007f]/.test(value)) {
        errors[key] = '不能包含空白或控制字符';
        continue;
      }
      try {
        const parsed = new URL(value);
        if (!parsed.hostname) throw new Error();
        if (!['http:', 'https:', 'socks5:', 'socks5h:'].includes(parsed.protocol)) throw new Error();
        if (parsed.username || parsed.password) throw new Error();
      } catch {
        errors[key] = '请输入有效的 http、https、socks5 或 socks5h 代理 URL，且不能包含账号密码';
      }
    }
    if (/[\u0000-\u001f\u007f]/.test(this.daemonProxyDraft.noProxy)) {
      errors.noProxy = '不能包含控制字符或换行';
    }
    this.daemonProxyErrors = errors;
    return !Object.values(errors).some(Boolean);
  }

  updateDaemonProxyDraft(key: 'httpProxy' | 'httpsProxy' | 'noProxy', value: string): void {
    this.daemonProxyDraft = { ...this.daemonProxyDraft, [key]: value };
    this.validateDaemonProxy();
  }

  private daemonProxyDraftValue(): { httpProxy: string; httpsProxy: string; noProxy: string } {
    return {
      httpProxy: this.daemonProxyDraft.httpProxy.trim(),
      httpsProxy: this.daemonProxyDraft.httpsProxy.trim(),
      noProxy: this.daemonProxyDraft.noProxy.trim(),
    };
  }

  async applyDaemonProxy(): Promise<void> {
    if (!this.validateDaemonProxy()) {
      this.toast.error('daemon 代理配置有误，请先修正标红字段');
      return;
    }
    const confirmed = await this.confirm.open({
      title: '覆写 Docker daemon 代理配置',
      message: '这会修改宿主机全局 Docker daemon 配置，影响所有容器后续的镜像拉取和构建。配置写入后需要重启 Docker daemon 才会生效。确认继续吗？',
      confirmText: '确认覆写',
      danger: true,
      critical: true,
    });
    if (!confirmed || this.daemonProxyBusy() || !this.daemonProxy()?.helperEnabled || !this.daemonProxy()?.writable) return;
    this.daemonProxyBusy.set(true);
    this.settings.applyDaemonProxy({ ...this.daemonProxyDraftValue(), hash: this.daemonProxyHash }).subscribe({
      next: response => {
        this.daemonProxyBusy.set(false);
        if (response.code !== 200 || !response.data) {
          this.toast.error(response.msg || '覆写 daemon 代理失败');
          return;
        }
        this.toast.success('daemon.json 已覆写；当前 Docker daemon 尚未更新，请按需重启使配置生效');
        this.loadDaemonProxy();
      },
      error: error => {
        this.daemonProxyBusy.set(false);
        this.toast.error(error?.error?.msg || '覆写 daemon 代理失败');
      },
    });
  }

  async restartDaemon(): Promise<void> {
    const confirmed = await this.confirm.open({
      title: '重启 Docker daemon',
      message: '这会短暂中断宿主机上所有容器的 Docker 管理操作，Docker Copilot 也可能暂时失联。确认重启吗？',
      confirmText: '确认重启',
      danger: true,
      critical: true,
    });
    if (!confirmed || this.daemonProxyBusy()) return;
    this.daemonProxyBusy.set(true);
    this.settings.restartDaemon().subscribe({
      next: response => {
        if (response.code !== 202 || !response.data) {
          this.daemonProxyBusy.set(false);
          this.toast.error(response.msg || '启动 Docker daemon 重启失败');
          return;
        }
        this.daemonRestart.set(response.data);
        this.daemonPollStartedAt = Date.now();
        this.daemonPollFailures = 0;
        this.pollDaemonRestart(response.data.operationID);
      },
      error: error => {
        this.daemonProxyBusy.set(false);
        this.toast.error(error?.error?.msg || '启动 Docker daemon 重启失败');
      },
    });
  }

  private pollDaemonRestart(operationID: string): void {
    if (Date.now() - this.daemonPollStartedAt > 120_000 || this.daemonPollFailures >= 20) {
      this.daemonProxyBusy.set(false);
      this.toast.error('无法确认 Docker daemon 最终状态，请手动检查 Docker 服务');
      return;
    }
    this.settings.getDaemonOperation(operationID).subscribe({
      next: response => {
        if (response.code !== 200 || !response.data) {
          this.daemonProxyBusy.set(false);
          this.toast.error(response.msg || '读取 Docker daemon 重启状态失败');
          return;
        }
        this.daemonRestart.set(response.data);
        if (response.data.status === 'restarting') {
          this.daemonPollTimer = setTimeout(() => this.pollDaemonRestart(operationID), 1500);
          return;
        }
        this.daemonProxyBusy.set(false);
        if (response.data.status === 'succeeded') {
          this.toast.success('Docker daemon 已重启并恢复');
          this.loadDaemonProxy();
        } else {
          this.toast.error(response.data.message || 'Docker daemon 重启失败');
        }
      },
      error: () => {
        this.daemonPollFailures++;
        this.daemonPollTimer = setTimeout(() => this.pollDaemonRestart(operationID), 2000);
      },
    });
  }

  addHubUrl(): void {
    if (this.hubUrlsDraft.length >= 20) {
      this.toast.error('最多 20 个加速源');
      return;
    }
    this.hubUrlsDraft = [...this.hubUrlsDraft, ''];
  }

  removeHubUrl(index: number): void {
    this.hubUrlsDraft = this.hubUrlsDraft.filter((_, i) => i !== index);
  }

  updateHubUrl(index: number, value: string): void {
    const next = [...this.hubUrlsDraft];
    next[index] = value;
    this.hubUrlsDraft = next;
  }

  resetHubUrls(): void {
    this.hubUrlsDraft = [...this.defaultHubUrls];
  }

  trackHubUrl(index: number): number {
    return index;
  }

  saveSettings(): void {
    if (!this.validateDaemonProxy()) {
      this.toast.error('daemon 代理配置有误，请先修正标红字段');
      return;
    }
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
        daemonProxyDraft: this.daemonProxyDraftValue(),
      })
      .subscribe({
        next: response => {
          if (response.code === 200 && response.data) {
            this.pullTimeoutDraft = response.data.pullTimeoutSec ?? pullTimeout;
            if (response.data.hubUrls) this.hubUrlsDraft = [...response.data.hubUrls];
            if (response.data.daemonProxyDraftConfigured) this.daemonProxyDraftConfigured = true;
          }
          this.finishSave(value, response.code !== 200, response.msg);
        },
        error: () => this.finishSave(value, true),
      });
  }

  private finishSave(value: number, failed: boolean, msg?: string): void {
    this.saving.set(false);
    if (failed) {
      this.toast.error(msg || '设置保存失败');
      return;
    }
    const runtime = this.settings.runtime();
    this.settings.setRuntime({
      updateInterval: this.updateDraft,
      backupInterval: this.backupDraft,
      logLevel: this.logDraft,
      retention: value,
    });
    this.closed.emit();
    const scope = [
      this.updateDraft !== runtime.updateInterval ? '更新检查频率已对后续自动检查生效' : '',
      this.backupDraft !== runtime.backupInterval ? '自动备份频率已对后续定时任务生效' : '',
      this.logDraft !== runtime.logLevel ? '日志级别已立即生效' : '',
      value !== runtime.retention ? '备份保留数量已对后续清理生效' : '',
      '镜像拉取超时和加速源已对新发起的任务生效',
    ].filter(Boolean).join('；');
    this.toast.success(msg || `设置已保存${scope ? '：' + scope : ''}`);
  }
}
