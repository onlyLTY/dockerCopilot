import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ComposeService, ComposeProject, ProjectsData } from '../../core/compose.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { ConfirmService } from '../../core/confirm.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { IconService } from '../../core/icon.service';
import { ResourceCardComponent } from '../../shared/resource-card/resource-card.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';

const defaultCompose = '';

@Component({
  selector: 'dc-compose',
  standalone: true,
  imports: [FormsModule, PageStateComponent, IconComponent, ResourceCardComponent, StatsComponent, PageHeadingComponent],
  templateUrl: './compose.component.html',
})
export class ComposeComponent {
  private readonly service = inject(ComposeService); private readonly icons = inject(IconService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService); private readonly confirm = inject(ConfirmService);
  // 项目数据、加载态来自服务常驻缓存
  readonly data = computed<ProjectsData | undefined>(() => {
    const value = this.service.cache.data();
    if (!value) return undefined;
    return {
      summary: value.summary || { total: 0, using: 0, stopped: 0, unused: 0, unknown: 0 },
      projects: Array.isArray(value.projects) ? value.projects.map(p => ({ ...p, files: Array.isArray(p.files) ? p.files : [], containers: Array.isArray(p.containers) ? p.containers : [], ports: Array.isArray(p.ports) ? p.ports : [] })) : [],
    };
  });
  readonly loading = this.service.cache.loading;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly filter = signal('all');
  readonly filteredProjects = computed(() => (this.data()?.projects || []).filter(p => this.filter() === 'all' || p.status === this.filter()));
  readonly stats = computed<readonly StatItem[]>(() => {
    const summary = this.data()?.summary;
    return [
      { key: 'all', value: summary?.total || 0, label: '项目总数' },
      { key: 'using', value: summary?.using || 0, label: '使用中', tone: 'green' },
      { key: 'stopped', value: summary?.stopped || 0, label: '已停止', tone: 'amber' },
      { key: 'unused', value: summary?.unused || 0, label: '未使用', tone: 'violet' },
      { key: 'unknown', value: summary?.unknown || 0, label: '未知', tone: 'red' },
    ];
  });
  readonly selectionMode = signal(false); readonly selected = signal<Set<string>>(new Set()); readonly cleanupBusy = signal(false);
  readonly editorProject = signal<ComposeProject | undefined>(undefined); readonly filename = signal(''); readonly content = signal(''); readonly version = signal(''); readonly message = signal(''); readonly messageType = signal(''); readonly pullImages = signal(false); readonly preview = signal<Record<string, any> | undefined>(undefined); readonly risks = signal<any[]>([]); readonly showCreate = signal(false); readonly creating = signal(false); projectName = ''; projectContent = defaultCompose; readonly createError = signal(''); readonly confirmDeploy = signal(false); readonly deployBusy = signal(false); readonly createPreview = signal<Record<string, any> | undefined>(undefined); readonly createValidated = signal(false);
  readonly composeDialog = computed(() => this.editorProject() ? 'edit' : this.showCreate() ? 'create' : 'closed');
  constructor() { this.service.ensureLoaded(); this.icons.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  selectFilter(key: string): void { this.filter.set(this.filter() === key || key === 'all' ? 'all' : key); this.selected.set(new Set()); }
  openEditor(p: ComposeProject) { const normalized = { ...p, files: Array.isArray(p.files) ? p.files : [], containers: Array.isArray(p.containers) ? p.containers : [], ports: Array.isArray(p.ports) ? p.ports : [] }; this.editorProject.set(normalized); this.preview.set(undefined); this.pullImages.set(false); this.message.set(''); this.messageType.set(''); const f = normalized.files.find(x => x.name === 'compose.yaml') || normalized.files[0]; if (f) this.openFile(f.name); }
  openFile(n: string) { const ep = this.editorProject(); if (!ep) return; this.filename.set(n); this.service.file(ep.id, n).subscribe({ next: r => r.code === 200 ? (this.content.set(r.data.content), this.version.set(r.data.version), this.message.set('')) : this.showError(r.msg, '读取文件'), error: e => this.showError(e.error?.msg || '读取文件失败', '读取文件') }); }
  save() { const ep = this.editorProject(); if (!ep) return; this.service.update(ep.id, this.filename(), this.content(), this.version()).subscribe({ next: r => r.code === 200 ? (this.version.set(r.data.version), this.message.set('已保存')) : this.showError(r.msg, '保存文件'), error: e => this.showError(e.error?.msg || '保存失败', '保存文件') }); }
  validate() {
    const ep = this.editorProject(); if (!ep) return;
    this.service.validate(ep.id, this.filename(), this.content()).subscribe({
      next: r => r.code === 200 ? (this.message.set('校验通过，包含 ' + (Array.isArray(r.data?.services) ? r.data.services.length : 0) + ' 个服务'), this.messageType.set('')) : this.showError(r.msg, '校验'),
      error: e => this.showError(e.error?.msg || '校验失败', '校验'),
    });
  }
  redeploy() {
    const ep = this.editorProject();
    if (!ep || this.deployBusy()) return;
    this.deployBusy.set(true);
    this.message.set('正在保存并校验配置…');
    this.service.update(ep.id, this.filename(), this.content(), this.version()).subscribe({
      next: saved => {
        if (saved.code !== 200) { this.deployBusy.set(false); this.showError(saved.msg, '重新部署'); return; }
        this.version.set(saved.data.version);
        this.service.validate(ep.id, this.filename(), this.content()).subscribe({
          next: validated => {
            if (validated.code !== 200) { this.deployBusy.set(false); this.showError(validated.msg, '重新部署'); return; }
            this.service.deployPreview(ep.id, this.filename()).subscribe({
              next: preview => {
                this.deployBusy.set(false);
                if (preview.code !== 200) { this.showError(preview.msg, '重新部署'); return; }
                this.askDeployment(ep, this.filename(), preview.data, this.pullImages());
              },
              error: e => { this.deployBusy.set(false); this.showError(e.error?.msg || '部署检查失败', '重新部署'); },
            });
          },
          error: e => { this.deployBusy.set(false); this.showError(e.error?.msg || '格式校验失败', '重新部署'); },
        });
      },
      error: e => { this.deployBusy.set(false); this.showError(e.error?.msg || '保存失败', '重新部署'); },
    });
  }
  private askDeployment(project: ComposeProject, filename: string, preview: Record<string, any>, pullImages = false) {
    const risks = Array.isArray(preview['risks']) ? preview['risks'] as any[] : [];
    this.confirm.open({
      title: '确认部署',
      message: `即将部署项目 ${project.name || project.id}。${risks.length ? `检测到 ${risks.length} 项风险，确认后继续。` : ''}${pullImages ? '将重新拉取项目镜像。' : ''}`,
      confirmText: '确定部署',
    }).then(ok => { if (ok) this.confirmDeployment(project, filename, preview, pullImages); });
  }
  private confirmDeployment(project: ComposeProject, filename: string, preview: Record<string, any>, pullImages = false) {
    if (this.deployBusy()) return;
    this.deployBusy.set(true);
    const risks = Array.isArray(preview['risks']) ? preview['risks'] as any[] : [];
    this.service.deploy(project.id, filename, String(preview['confirmToken']), risks.length > 0, pullImages).subscribe({
      next: r => {
        this.deployBusy.set(false);
        if (r.code !== 200) { this.showError(r.msg, '部署'); return; }
        const taskID = (r.data as any)?.['taskID'];
        if (taskID) {
          this.tasks.track(String(taskID), '部署 ' + (project.name || project.id), true);
          this.closeEditor();
        } else {
          this.toast.success('部署命令已完成');
        }
      },
      error: e => { this.deployBusy.set(false); this.showError(e.error?.msg || '部署失败', '部署'); },
    });
  }
  newProject() { this.projectName = ''; this.projectContent = defaultCompose; this.createError.set(''); this.createValidated.set(false); this.showCreate.set(true); }
  validateCreate() { const n = this.projectName.trim(); if (!n) { this.notifyComposeError('校验', '请输入项目名称'); return; } this.service.validate('', 'compose.yaml', this.projectContent).subscribe({ next: r => { if (r.code === 200) { this.createValidated.set(true); this.createError.set('格式校验通过，包含 ' + (Array.isArray(r.data?.services) ? r.data.services.length : 0) + ' 个服务'); } else this.notifyComposeError('校验', r.msg); }, error: e => this.notifyComposeError('校验', e.error?.msg || '格式校验失败') }); }
  deployNewProject() { this.createProject(true); }
  createProject(deployAfterCreate = false) {
    const n = this.projectName.trim(); if (!n) { this.notifyComposeError('创建', '请输入项目名称'); return; }
    this.creating.set(true);
    this.service.createProject(n, 'compose.yaml', this.projectContent).subscribe({
      next: r => {
        this.creating.set(false);
        if (r.code !== 200) { this.notifyComposeError('创建', r.msg); return; }
        if (!deployAfterCreate) { this.closeCreate(); this.toast.success('项目已创建'); return; }
        this.createAndPreview(r.data.projectId);
      },
      error: e => { this.creating.set(false); this.notifyComposeError('创建', e.error?.msg || '创建项目失败'); },
    });
  }
  private createAndPreview(projectId: string) {
    this.service.projects().subscribe({
      next: projects => {
        const project = projects.data?.projects?.find(item => item.id === projectId);
        if (!project) { this.notifyComposeError('读取项目', '项目已创建，但读取项目详情失败'); return; }
        const file = project.files.find(item => item.name === 'compose.yaml') || project.files[0];
        if (!file) { this.notifyComposeError('读取项目', '项目已创建，但没有找到 Compose 文件'); return; }
        this.service.deployPreview(project.id, file.name).subscribe({
          next: preview => {
            if (preview.code !== 200) { this.notifyComposeError('部署预览', preview.msg || '部署预览失败'); return; }
            this.closeCreate();
            this.askDeployment(project, file.name, preview.data);
          },
          error: e => this.notifyComposeError('部署预览', e.error?.msg || '部署预览失败'),
        });
      },
      error: () => this.notifyComposeError('读取项目', '项目已创建，但读取项目详情失败'),
    });
  }
  enterSelection() { this.selectionMode.set(true); }
  exitSelection() { this.selectionMode.set(false); this.selected.set(new Set()); }
  isSelected(id: string) { return this.selected().has(id); }
  toggleSelect(id: string) { const next = new Set(this.selected()); next.has(id) ? next.delete(id) : next.add(id); this.selected.set(next); }
  readonly allSelected = computed(() => (this.data()?.projects || []).length > 0 && (this.data()?.projects || []).every(p => this.selected().has(p.id)));
  toggleAll() { const projects = this.data()?.projects || []; const next = new Set(this.selected()); if (this.allSelected()) projects.forEach(p => next.delete(p.id)); else projects.forEach(p => next.add(p.id)); this.selected.set(next); }
  bulkCleanup() {
    const projects = (this.data()?.projects || []).filter(p => this.selected().has(p.id) && p.status === 'unused');
    if (!projects.length || this.cleanupBusy()) return;
    this.confirm.open({ title: '清理未使用项目', message: `确定清理选中的 ${projects.length} 个未使用项目吗？`, confirmText: '确认清理', danger: true }).then(ok => { if (ok) this.runBulkCleanup(projects); });
  }
  private runBulkCleanup(projects: ComposeProject[]) {
    this.cleanupBusy.set(true); let remaining = projects.length; let failed = 0;
    projects.forEach(project => this.service.cleanupPreview(project.id).subscribe({
      next: preview => {
        if (preview.code !== 200 || !preview.data?.['previewToken']) { failed++; if (!--remaining) this.finishCleanup(projects.length, failed); return; }
        this.service.cleanup(project.id, String(preview.data['previewToken'])).subscribe({ next: r => { if (r.code !== 200) failed++; if (!--remaining) this.finishCleanup(projects.length, failed); }, error: () => { failed++; if (!--remaining) this.finishCleanup(projects.length, failed); } });
      }, error: () => { failed++; if (!--remaining) this.finishCleanup(projects.length, failed); },
    }));
  }
  private finishCleanup(total: number, failed: number) { this.cleanupBusy.set(false); this.exitSelection(); this.service.refresh(); if (failed) this.toast.error(`项目清理完成 ${total - failed} 个，失败 ${failed} 个`); else this.toast.success(`已清理 ${total} 个项目`); }
  closeEditor(e?: Event) { if (!e || e.target === e.currentTarget) { this.editorProject.set(undefined); } }
  closeCreate(e?: Event) { if (!e || e.target === e.currentTarget) { this.showCreate.set(false); this.creating.set(false); this.createError.set(''); this.createValidated.set(false); this.projectName = ''; this.projectContent = defaultCompose; } }
  private notifyComposeError(operation: string, detail: string): void {
    const message = detail || '未知错误';
    this.message.set('');
    this.messageType.set('error');
    this.createError.set('');
    this.toast.error(`Compose ${operation}失败`, message);
  }
  private showError(detail: string, operation = '操作'): void { this.notifyComposeError(operation, detail); }
  statusLabel(s: string) { return ({ using: '使用中', stopped: '已停止', unused: '未使用', unknown: '未知' } as Record<string, string>)[s] || s; }
  icon(project: ComposeProject) { return this.icons.resolve(project.image || '', this.iconMap()); }
  fallback(event: Event) { (event.target as HTMLImageElement).src = this.icons.actionIcon('images'); }
}
