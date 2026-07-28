import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BackupService } from '../../core/backup.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-backups',
  standalone: true,
  imports: [CommonModule, FormsModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './backups.component.html',
})
export class BackupsComponent {
  private readonly service = inject(BackupService); files: string[] = []; retention = 10; retentionDraft = 10; error = ''; settingsError = ''; saving = false; loading = false; showSettings = false;
  get jsonCount() { return this.files.filter(x => x.endsWith('.json')).length; } get yamlCount() { return this.files.filter(x => x.endsWith('.yaml') || x.endsWith('.yml')).length; }
  get groups() { const map = new Map<string, string[]>(); this.files.forEach(x => { const d = this.date(x); if (!map.has(d)) map.set(d, []); map.get(d)!.push(x); }); return Array.from(map, ([date, files]) => ({ date, files })); }
  constructor() { this.load(); this.service.settings().subscribe({ next: r => { if (r.code === 200) { this.retention = r.data.retention; this.retentionDraft = this.retention; } } }); }
  load() { this.loading = true; this.error = ''; this.service.list().subscribe({ next: r => { this.loading = false; if (r.code === 200) this.files = r.data || []; else this.error = r.msg; }, error: e => { this.loading = false; this.error = e.error?.msg || '读取备份失败'; } }); }
  openSettings() { this.retentionDraft = this.retention; this.settingsError = ''; this.showSettings = true; }
  closeSettings(e?: Event) { if (!e || e.target === e.currentTarget) { this.showSettings = false; this.settingsError = ''; } }
  saveSettings() { const value = Number(this.retentionDraft); if (!Number.isInteger(value) || value < 1 || value > 100) { this.settingsError = '保留数量必须是 1-100 的整数'; return; } this.saving = true; this.service.updateSettings(value).subscribe({ next: r => { this.saving = false; if (r.code === 200) { this.retention = r.data.retention; this.retentionDraft = this.retention; this.showSettings = false; this.load(); } else this.settingsError = r.msg; }, error: e => { this.saving = false; this.settingsError = e.error?.msg || '保存设置失败'; } }); }
  create(t: 'json' | 'yaml') { const req = t === 'json' ? this.service.createJson() : this.service.createYaml(); req.subscribe({ next: r => r.code === 200 ? this.load() : this.error = r.msg, error: e => this.error = e.error?.msg || '创建备份失败' }); }
  restore(f: string) { if (!confirm('恢复备份 ' + f + '？')) return; this.service.restore(f).subscribe({ next: r => this.error = r.code === 200 ? '恢复任务已提交' : r.msg }); }
  remove(f: string) { if (!confirm('删除备份 ' + f + '？')) return; this.service.remove(f).subscribe({ next: r => r.code === 200 ? this.load() : this.error = r.msg }); }
  date(f: string) { const m = f.match(/(\d{4}-\d{2}-\d{2})/); return m?.[1] || '其他日期'; } base(f: string) { return f.replace(/\.(json|ya?ml)$/, ''); } type(f: string) { return f.endsWith('.json') ? 'JSON' : 'YAML'; }
}
