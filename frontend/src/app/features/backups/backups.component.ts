import { Component, inject, computed } from '@angular/core';
import { BackupService } from '../../core/backup.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';

@Component({
  selector: 'dc-backups',
  standalone: true,
  imports: [PageStateComponent, IconComponent],
  templateUrl: './backups.component.html',
})
export class BackupsComponent {
  private readonly service = inject(BackupService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService); private readonly confirm = inject(ConfirmService);
  readonly files = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly jsonCount = computed(() => this.files().filter(x => this.extension(x) === '.json').length);
  readonly yamlCount = computed(() => this.files().filter(x => ['.yaml', '.yml'].includes(this.extension(x))).length);
  readonly groups = computed(() => { const map = new Map<string, string[]>(); this.files().forEach(x => { const d = this.date(x); if (!map.has(d)) map.set(d, []); map.get(d)!.push(x); }); return Array.from(map, ([date, files]) => ({ date, files })); });
  constructor() { this.service.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  create(t: 'json' | 'yaml') { const req = t === 'json' ? this.service.createJson() : this.service.createYaml(); req.subscribe({ next: r => { if (r.code === 200) { this.toast.success('备份已创建', `${t.toUpperCase()} 备份已创建`); } else this.toast.error('创建备份失败', r.msg || '未知错误'); }, error: e => this.toast.error('创建备份失败', e.error?.msg || e.message || '请求错误') }); }
  async restore(f: string) { if (!(await this.confirm.open({ title: '恢复备份', message: `恢复备份 ${f}？`, confirmText: '恢复' }))) return; this.service.restore(f).subscribe({ next: r => { if (r.code === 200 && r.data?.taskID) { this.tasks.track(r.data.taskID, '恢复 ' + this.date(f), true); } else if (r.code === 200) this.toast.info('恢复任务已提交', '恢复任务已提交'); else this.toast.error('恢复失败', r.msg || '未知错误'); }, error: e => this.toast.error('恢复失败', e.error?.msg || e.message || '请求错误') }); }
  async remove(f: string) { if (!(await this.confirm.open({ title: '删除备份', message: `删除备份 ${f}？`, confirmText: '删除', danger: true }))) return; this.service.remove(f).subscribe({ next: r => { if (r.code === 200) this.toast.success('备份已删除', f); else this.toast.error('删除失败', r.msg || '未知错误'); }, error: e => this.toast.error('删除失败', e.error?.msg || e.message || '请求错误') }); }
  date(f: string) { const m = f.match(/(\d{4}-\d{2}-\d{2})/) || f.match(/(\d{4})(\d{2})(\d{2})/); return m ? (m[1].includes('-') ? m[1] : `${m[1]}-${m[2]}-${m[3]}`) : '其他日期'; }
  type(f: string) { return this.extension(f) === '.json' ? 'JSON' : 'YAML'; }
  extension(f: string) { const dot = f.lastIndexOf('.'); return dot >= 0 ? f.slice(dot).toLowerCase() : ''; }
}