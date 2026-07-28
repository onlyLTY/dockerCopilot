import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BackupService } from '../../core/backup.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-backups',
  standalone: true,
  imports: [FormsModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './backups.component.html',
})
export class BackupsComponent {
  private readonly service = inject(BackupService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService);
  // 数据、加载态、错误态来自服务常驻缓存
  readonly files = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly retention = computed(() => this.service.settingsCache.data()?.retention ?? 10);
  retentionDraft = 10; readonly settingsError = signal<string>(''); readonly saving = signal<boolean>(false); readonly showSettings = signal<boolean>(false);
  readonly jsonCount = computed(() => this.files().filter(x => x.endsWith('.json')).length); readonly yamlCount = computed(() => this.files().filter(x => x.endsWith('.yaml') || x.endsWith('.yml')).length);
  readonly groups = computed(() => { const map = new Map<string, string[]>(); this.files().forEach(x => { const d = this.date(x); if (!map.has(d)) map.set(d, []); map.get(d)!.push(x); }); return Array.from(map, ([date, files]) => ({ date, files })); });
  constructor() { this.service.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  openSettings() { this.retentionDraft = this.retention(); this.settingsError.set(''); this.showSettings.set(true); }
  closeSettings(e?: Event) { if (!e || e.target === e.currentTarget) { this.showSettings.set(false); this.settingsError.set(''); } }
  saveSettings() { const value = Number(this.retentionDraft); if (!Number.isInteger(value) || value < 1 || value > 100) { this.settingsError.set('保留数量必须是 1-100 的整数'); return; } this.saving.set(true); this.service.updateSettings(value).subscribe({ next: r => { this.saving.set(false); if (r.code === 200) { this.retentionDraft = r.data.retention; this.showSettings.set(false); this.toast.success('保留设置已保存'); } else this.settingsError.set(r.msg); }, error: e => { this.saving.set(false); this.settingsError.set(e.error?.msg || '保存设置失败'); } }); }
  create(t: 'json' | 'yaml') { const req = t === 'json' ? this.service.createJson() : this.service.createYaml(); req.subscribe({ next: r => { if (r.code === 200) { this.toast.success(`${t.toUpperCase()} 备份已创建`); } else this.toast.error(`创建备份失败：${r.msg || '未知错误'}`); }, error: e => this.toast.error(`创建备份失败：${e.error?.msg || e.message || '请求错误'}`) }); }
  restore(f: string) { if (!confirm('恢复备份 ' + f + '？')) return; this.service.restore(f).subscribe({ next: r => { if (r.code === 200 && r.data?.taskID) { this.tasks.track(r.data.taskID, '恢复 ' + this.base(f), true); } else if (r.code === 200) { this.toast.info('恢复任务已提交'); } else this.toast.error(`恢复失败：${r.msg || '未知错误'}`); }, error: e => this.toast.error(`恢复失败：${e.error?.msg || e.message || '请求错误'}`) }); }
  remove(f: string) { if (!confirm('删除备份 ' + f + '？')) return; this.service.remove(f).subscribe({ next: r => { if (r.code === 200) { this.toast.success('备份已删除'); } else this.toast.error(`删除失败：${r.msg || '未知错误'}`); }, error: e => this.toast.error(`删除失败：${e.error?.msg || e.message || '请求错误'}`) }); }
  date(f: string) { const m = f.match(/(\d{4}-\d{2}-\d{2})/); return m?.[1] || '其他日期'; } base(f: string) { const m = f.match(/^(.*?\d{4}-\d{2}-\d{2})/); return m?.[1] || f.replace(/\.(json|ya?ml)$/, ''); } type(f: string) { return f.endsWith('.json') ? 'JSON' : 'YAML'; }
}
