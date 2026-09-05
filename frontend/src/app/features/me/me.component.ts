import { Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { SettingsService } from '../../core/settings.service';
import { VersionService, VersionInfo } from '../../core/version.service';
import { ConfirmService } from '../../core/confirm.service';
import { IconComponent } from '../../shared/icon/icon.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { SettingsDialogComponent } from './components/settings-dialog/settings-dialog.component';
import { LogDialogComponent } from './components/log-dialog/log-dialog.component';

type UpdatePhase =
  | 'idle' // 未检查
  | 'checking' // 检查更新中
  | 'latest' // 已是最新
  | 'available' // 发现新版本
  | 'downloading' // 更新包下载中（PUT 进行中，可能数分钟）
  | 'countdown' // 下载完成，等待服务自动退出（约 10 秒）
  | 'restarting' // 轮询等待服务恢复
  | 'success' // 确认新版本已上线，即将刷新页面
  | 'stale' // 服务已恢复但版本号未变（旧镜像的 version 文件未随自更新切换）
  | 'failed'; // 失败

/** 重启轮询：等待服务恢复的超时（秒），超过视为失败 */
const RESTART_TIMEOUT_S = 180;
/** 服务恢复但版本号未变时，继续轮询到该秒数后判定为版本号未随更新切换（秒） */
const RESTART_STALE_S = 60;

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
  private readonly confirm = inject(ConfirmService);
  private readonly destroyRef = inject(DestroyRef);

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

  readonly updatePhase = signal<UpdatePhase>('idle');
  readonly remoteVersion = signal('');
  readonly updateError = signal('');
  readonly waitSeconds = signal(0);
  /** 不可打断的阶段：检查 / 下载 / 等待重启 */
  readonly busyUpdate = computed(() =>
    ['checking', 'downloading', 'countdown', 'restarting'].includes(
      this.updatePhase(),
    ),
  );

  private tickers: ReturnType<typeof setInterval>[] = [];
  private readonly unloadGuard = (event: BeforeUnloadEvent) => {
    event.preventDefault();
    event.returnValue = '';
  };

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
    this.destroyRef.onDestroy(() => {
      this.clearTickers();
      this.setUnloadGuard(false);
    });
  }

  async checkUpdate(): Promise<void> {
    if (this.busyUpdate()) return;
    this.updatePhase.set('checking');
    this.updateError.set('');
    this.versions.remote().subscribe({
      next: r => {
        const remote = r?.data?.remoteVersion || '';
        if (r?.code === 200 && remote && remote !== this.version().version) {
          this.remoteVersion.set(remote);
          this.updatePhase.set('available');
        } else if (r?.code === 200) {
          this.updatePhase.set('latest');
        } else {
          this.finishFailed(r?.msg || '检查更新失败');
        }
      },
      error: e => this.finishFailed(this.describeError(e, '检查更新失败')),
    });
  }

  async startUpdate(): Promise<void> {
    if (this.busyUpdate()) return;
    const ok = await this.confirm.open({
      title: '更新程序',
      message: `将下载并安装 ${this.remoteVersion()}，完成后服务自动重启，期间页面会短暂失去连接，重启完成后自动刷新。确定继续吗？`,
      confirmText: '开始更新',
    });
    if (!ok) return;
    this.updatePhase.set('downloading');
    this.updateError.set('');
    this.waitSeconds.set(0);
    this.setUnloadGuard(true);
    this.every(1000, () => this.waitSeconds.update(s => s + 1));
    this.versions.updateProgram().subscribe({
      next: r => {
        if (r?.code === 200) this.beginRestartWatch();
        else this.finishFailed(r?.msg || '更新失败');
      },
      error: e => this.finishFailed(this.describeError(e, '下载更新包失败')),
    });
  }

  reload(): void {
    window.location.reload();
  }

  /** 下载成功后：等服务退出（10 秒宽限），再轮询 /healthz 直到新版本上线 */
  private beginRestartWatch(): void {
    this.clearTickers();
    let left = 10;
    this.updatePhase.set('countdown');
    this.every(1000, () => {
      left -= 1;
      if (left <= 0) this.pollRestart();
    });
  }

  private pollRestart(): void {
    this.clearTickers();
    this.updatePhase.set('restarting');
    this.waitSeconds.set(0);
    this.every(1000, () => this.waitSeconds.update(s => s + 1));
    this.pollOnce();
    this.every(2000, () => this.pollOnce());
  }

  private pollOnce(): void {
    if (this.updatePhase() !== 'restarting') return;
    this.versions.health().subscribe({
      next: health => {
        if (this.updatePhase() !== 'restarting') return;
        if (health?.version && health.version === this.remoteVersion()) {
          this.succeed();
        } else if (this.waitSeconds() >= RESTART_STALE_S) {
          // 服务已稳定恢复但版本号未变：多为旧镜像的 version 文件未随自更新切换
          this.clearTickers();
          this.setUnloadGuard(false);
          this.updatePhase.set('stale');
        }
      },
      error: () => {
        // 连接被拒 = 服务正在重启，继续等待；超时则报失败
        if (
          this.updatePhase() === 'restarting' &&
          this.waitSeconds() >= RESTART_TIMEOUT_S
        ) {
          this.finishFailed('服务长时间未恢复，请检查容器状态或查看服务日志');
        }
      },
    });
  }

  private succeed(): void {
    this.clearTickers();
    this.setUnloadGuard(false);
    this.updatePhase.set('success');
    setTimeout(() => window.location.reload(), 2500);
  }

  private finishFailed(message: string): void {
    this.clearTickers();
    this.setUnloadGuard(false);
    this.updateError.set(message);
    this.updatePhase.set('failed');
  }

  private clearTickers(): void {
    this.tickers.forEach(t => clearInterval(t));
    this.tickers = [];
  }

  private every(ms: number, fn: () => void): void {
    this.tickers.push(setInterval(fn, ms));
  }

  private setUnloadGuard(on: boolean): void {
    if (on) window.addEventListener('beforeunload', this.unloadGuard);
    else window.removeEventListener('beforeunload', this.unloadGuard);
  }

  private describeError(e: unknown, fallback: string): string {
    const err = e as HttpErrorResponse | null;
    const body = err?.error as { msg?: string; message?: string } | string | null;
    const text =
      (typeof body === 'string' && body.trim()) ||
      (typeof body === 'object' && body?.msg) ||
      '';
    return text || fallback;
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
