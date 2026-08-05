import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SettingsService } from '../../core/settings.service';
import { ToastService } from '../../core/toast.service';
import { SectionToolbarComponent } from '../../shared/section-toolbar/section-toolbar.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { FormSelectComponent, FormSelectOption } from '../../shared/form-select/form-select.component';
import { actionErrorMessage, runAction } from '../../core/run-action';

/**
  * 设置页：提供定时任务配置（预设下拉）。
 * - 更新检查频率：定时检查镜像是否有新版本
 * - 自动备份频率：定时创建容器备份（JSON + YAML）
 * 频率由后端持久化，保存后立即按新频率重新调度对应定时任务。
 */
@Component({
  selector: 'dc-settings',
  standalone: true,
  imports: [FormsModule, SectionToolbarComponent, PageHeadingComponent, FormSelectComponent],
  templateUrl: './settings.component.html',
})
export class SettingsComponent {
  private readonly service = inject(SettingsService); private readonly toast = inject(ToastService);

  // 更新检查频率
  readonly updateOptions = signal<string[]>([]);
  readonly updateInterval = signal<string>('');
  readonly updateSaving = signal<boolean>(false);
  updateDraft = '';
  readonly updateDirty = computed(() => this.updateDraft !== this.updateInterval());

  // 自动备份频率
  readonly backupOptions = signal<string[]>([]);
  readonly backupInterval = signal<string>('');
  readonly backupSaving = signal<boolean>(false);
  backupDraft = '';
  readonly backupDirty = computed(() => this.backupDraft !== this.backupInterval());

  readonly loading = signal<boolean>(false);
  readonly updateSelectOptions = computed<FormSelectOption[]>(() => this.updateOptions().map(value => ({ value, label: this.updateLabel(value) })));
  readonly backupSelectOptions = computed<FormSelectOption[]>(() => this.backupOptions().map(value => ({ value, label: this.backupLabel(value) })));

  private readonly updateLabels: Record<string, string> = {
    off: '关闭（仅手动检查）', '30m': '每 30 分钟', '1h': '每小时', '6h': '每 6 小时', '12h': '每 12 小时', '24h': '每天一次',
  };
  private readonly backupLabels: Record<string, string> = {
    off: '关闭（仅手动备份）', '6h': '每 6 小时', '12h': '每 12 小时', '24h': '每天一次', '7d': '每周一次',
  };
  updateLabel(key: string) { return this.updateLabels[key] || key; }
  backupLabel(key: string) { return this.backupLabels[key] || key; }

  constructor() { this.load(); }
  load() {
    this.loading.set(true);
    this.service.getUpdateSettings().subscribe({
      next: r => {
        this.loading.set(false);
        if (r.code === 200) { this.updateOptions.set(r.data.options || []); this.updateInterval.set(r.data.interval); this.updateDraft = r.data.interval; }
        else this.toast.error(`读取更新检查设置失败：${r.msg || '未知错误'}`);
      },
      error: e => { this.loading.set(false); this.toast.error(`读取更新检查设置失败：${e.error?.msg || e.message || '请求错误'}`); },
    });
    this.service.getBackupSettings().subscribe({
      next: r => {
        if (r.code === 200) { this.backupOptions.set(r.data.options || []); this.backupInterval.set(r.data.interval); this.backupDraft = r.data.interval; }
        else this.toast.error(`读取自动备份设置失败：${r.msg || '未知错误'}`);
      },
      error: e => this.toast.error(`读取自动备份设置失败：${e.error?.msg || e.message || '请求错误'}`),
    });
  }
  saveUpdate() {
    if (!this.updateDirty() || this.updateSaving()) return;
    runAction({
      request: this.service.setUpdateInterval(this.updateDraft),
      onStart: () => this.updateSaving.set(true),
      onFinally: () => this.updateSaving.set(false),
      onSuccess: r => {
        this.updateInterval.set(r.data.interval);
        this.updateDraft = r.data.interval;
        this.toast.success('更新检查频率已保存');
      },
      onBizError: r => this.toast.error(`保存失败：${r.msg || '未知错误'}`),
      onHttpError: e => this.toast.error(`保存失败：${actionErrorMessage(e)}`),
    });
  }
  saveBackup() {
    if (!this.backupDirty() || this.backupSaving()) return;
    runAction({
      request: this.service.setBackupInterval(this.backupDraft),
      onStart: () => this.backupSaving.set(true),
      onFinally: () => this.backupSaving.set(false),
      onSuccess: r => {
        this.backupInterval.set(r.data.interval);
        this.backupDraft = r.data.interval;
        this.toast.success('自动备份频率已保存');
      },
      onBizError: r => this.toast.error(`保存失败：${r.msg || '未知错误'}`),
      onHttpError: e => this.toast.error(`保存失败：${actionErrorMessage(e)}`),
    });
  }
}
