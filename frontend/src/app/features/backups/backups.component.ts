import { Component, inject, computed, signal } from '@angular/core';
import { BackupService } from '../../core/backup.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';
import { actionErrorMessage, runAction } from '../../core/run-action';

@Component({
  selector: 'dc-backups',
  standalone: true,
  imports: [PageStateComponent, IconComponent, StatsComponent, PageHeadingComponent],
  templateUrl: './backups.component.html',
})
export class BackupsComponent {
  private readonly service = inject(BackupService);
  private readonly toast = inject(ToastService);
  private readonly tasks = inject(TaskService);
  private readonly confirm = inject(ConfirmService);
  readonly files = computed(() => this.service.cache.data() || []);
  readonly filter = signal('all');
  readonly filteredFiles = computed(() => this.files().filter(f => this.filter() === 'all' || (this.filter() === 'json' ? this.extension(f) === '.json' : ['.yaml', '.yml'].includes(this.extension(f)))));
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly jsonCount = computed(() => this.files().filter(x => this.extension(x) === '.json').length);
  readonly yamlCount = computed(() => this.files().filter(x => ['.yaml', '.yml'].includes(this.extension(x))).length);
  readonly creating = signal(false);
  readonly stats = computed<readonly StatItem[]>(() => [
    { key: 'all', value: this.files().length, label: '总备份数' },
    { key: 'json', value: this.jsonCount(), label: 'JSON 备份', tone: 'blue' },
    { key: 'yaml', value: this.yamlCount(), label: 'YAML 备份', tone: 'violet' },
  ]);
  readonly groups = computed(() => {
    const map = new Map<string, string[]>();
    this.filteredFiles().forEach(x => {
      const d = this.date(x);
      if (!map.has(d)) map.set(d, []);
      map.get(d)!.push(x);
    });
    return Array.from(map, ([date, files]) => ({ date, files }));
  });
  constructor() { this.service.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  selectFilter(key: string): void { this.filter.set(this.filter() === key || key === 'all' ? 'all' : key); }

  create(t: 'json' | 'yaml') {
    if (this.creating()) return;
    const req = t === 'json' ? this.service.createJson() : this.service.createYaml();
    runAction({
      request: req,
      onStart: () => this.creating.set(true),
      onFinally: () => this.creating.set(false),
      onSuccess: () => this.toast.success('备份已创建', `${t.toUpperCase()} 备份已创建`),
      onBizError: r => this.toast.error('创建备份失败', r.msg || '未知错误'),
      onHttpError: e => this.toast.error('创建备份失败', actionErrorMessage(e)),
    });
  }

  async restore(f: string) {
    if (!(await this.confirm.open({ title: '恢复备份', message: `恢复备份 ${f}？`, confirmText: '恢复' }))) return;
    runAction({
      request: this.service.restore(f),
      onSuccess: r => {
        const taskID = (r.data as { taskID?: string } | undefined)?.taskID;
        if (taskID) this.tasks.track(taskID, '恢复 ' + this.date(f), true);
        else this.toast.info('恢复任务已提交');
      },
      onBizError: r => this.toast.error('恢复失败', r.msg || '未知错误'),
      onHttpError: e => this.toast.error('恢复失败', actionErrorMessage(e)),
    });
  }

  async remove(f: string) {
    if (!(await this.confirm.open({ title: '删除备份', message: `删除备份 ${f}？`, confirmText: '删除', danger: true }))) return;
    runAction({
      request: this.service.remove(f),
      onSuccess: () => this.toast.success('备份已删除', f),
      onBizError: r => this.toast.error('删除失败', r.msg || '未知错误'),
      onHttpError: e => this.toast.error('删除失败', actionErrorMessage(e)),
    });
  }

  date(f: string) {
    const m = f.match(/(\d{4}-\d{2}-\d{2})/) || f.match(/(\d{4})(\d{2})(\d{2})/);
    return m ? (m[1].includes('-') ? m[1] : `${m[1]}-${m[2]}-${m[3]}`) : '其他日期';
  }
  type(f: string) { return this.extension(f) === '.json' ? 'JSON' : 'YAML'; }
  extension(f: string) { const dot = f.lastIndexOf('.'); return dot >= 0 ? f.slice(dot).toLowerCase() : ''; }
}
