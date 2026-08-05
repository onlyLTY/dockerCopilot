import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
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
import { ModalHeadingComponent } from '../../shared/modal-heading/modal-heading.component';
import { actionErrorMessage, runAction } from '../../core/run-action';

const defaultCompose = '';

@Component({
  selector: 'dc-compose',
  standalone: true,
  imports: [FormsModule, PageStateComponent, IconComponent, ResourceCardComponent, StatsComponent, PageHeadingComponent, ModalHeadingComponent],
  templateUrl: './compose.component.html',
})
export class ComposeComponent {
  private readonly service = inject(ComposeService); private readonly icons = inject(IconService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService); private readonly confirm = inject(ConfirmService); private readonly sanitizer = inject(DomSanitizer);
  // 行号：基于编辑器内容行数生成，外层 textarea 滚动时同步 gutter 的 scrollTop
  readonly lineCount = computed(() => Math.max(1, (this.content() || '').split('\n').length));
  readonly lineNumbers = computed<SafeHtml>(() => this.sanitizer.bypassSecurityTrustHtml(this.buildLines(this.lineCount())));
  readonly createLineCount = computed(() => Math.max(1, (this.projectContent() || '').split('\n').length));
  readonly createLineNumbers = computed<SafeHtml>(() => this.sanitizer.bypassSecurityTrustHtml(this.buildLines(this.createLineCount())));
  private buildLines(n: number): string {
    let out = '';
    for (let i = 1; i <= n; i++) out += `<span>${i}</span>`;
    return out;
  }
  onEditorScroll(e: Event) { this.syncGutter(e); }
  onCreateScroll(e: Event) { this.syncGutter(e); }
  private syncGutter(e: Event) {
    const ta = e.target as HTMLTextAreaElement;
    const gutter = ta.parentElement?.querySelector('.code-gutter') as HTMLElement | null;
    if (gutter) gutter.scrollTop = ta.scrollTop;
  }
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
  readonly error = this.service.cache.error;
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
  readonly editorProject = signal<ComposeProject | undefined>(undefined); readonly filename = signal(''); readonly content = signal(''); readonly version = signal(''); readonly fileLoading = signal(false); readonly fileError = signal(''); readonly message = signal(''); readonly messageType = signal(''); readonly pullImages = signal(false); readonly preview = signal<Record<string, any> | undefined>(undefined); readonly risks = signal<any[]>([]); readonly showCreate = signal(false); readonly creating = signal(false); readonly projectName = signal(''); readonly projectContent = signal(defaultCompose); readonly createError = signal(''); readonly createErrorType = signal<'error' | 'success' | ''>(''); readonly confirmDeploy = signal(false); readonly deployBusy = signal(false); readonly createPreview = signal<Record<string, any> | undefined>(undefined); readonly createValidated = signal(false);
  readonly composeDialog = computed(() => this.editorProject() ? 'edit' : this.showCreate() ? 'create' : 'closed');
  constructor() { this.service.ensureLoaded(); this.icons.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  selectFilter(key: string): void { this.filter.set(this.filter() === key || key === 'all' ? 'all' : key); this.selected.set(new Set()); }
  openEditor(p: ComposeProject) { const normalized = { ...p, files: Array.isArray(p.files) ? p.files : [], containers: Array.isArray(p.containers) ? p.containers : [], ports: Array.isArray(p.ports) ? p.ports : [] }; this.editorProject.set(normalized); this.filename.set(''); this.content.set(''); this.version.set(''); this.fileLoading.set(false); this.fileError.set(''); this.preview.set(undefined); this.pullImages.set(false); this.message.set(''); this.messageType.set(''); const f = normalized.files.find(x => x.name === 'compose.yaml') || normalized.files[0]; if (f) this.openFile(f.name); else this.fileError.set('未找到 Compose 文件'); }
  openFile(n: string) { const ep = this.editorProject(); if (!ep) return; this.filename.set(n); this.content.set(''); this.version.set(''); this.fileError.set(''); this.fileLoading.set(true); this.service.file(ep.id, n).subscribe({ next: r => { if (this.editorProject()?.id !== ep.id || this.filename() !== n) return; this.fileLoading.set(false); if (r.code === 200) { this.content.set(r.data.content); this.version.set(r.data.version); this.message.set(''); } else { this.fileError.set(r.msg || '读取文件失败'); } }, error: e => { if (this.editorProject()?.id !== ep.id || this.filename() !== n) return; this.fileLoading.set(false); this.fileError.set(e.error?.msg || '读取文件失败'); } }); }
  save() {
    const ep = this.editorProject(); if (!ep) return;
    runAction({
      request: this.service.update(ep.id, this.filename(), this.content(), this.version()),
      onSuccess: r => { this.version.set((r.data as { version: string }).version); this.message.set('已保存'); this.messageType.set(''); },
      onBizError: r => this.showError(r.msg, '保存文件'),
      onHttpError: e => this.showError(actionErrorMessage(e, '保存失败'), '保存文件'),
    });
  }
  validate() {
    const ep = this.editorProject(); if (!ep) return;
    this.content.set(this.normalizeCompose(this.content()));
    runAction({
      request: this.service.validate(ep.id, this.filename(), this.content()),
      onSuccess: r => {
        const n = Array.isArray((r.data as { services?: unknown[] } | undefined)?.services) ? (r.data as { services: unknown[] }).services.length : 0;
        this.message.set('校验通过，包含 ' + n + ' 个服务');
        this.messageType.set('');
      },
      onBizError: r => this.showError(r.msg, '校验'),
      onHttpError: e => this.showError(actionErrorMessage(e, '校验失败'), '校验'),
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
    const risks = Array.isArray(preview['risks']) ? preview['risks'] as any[] : [];
    runAction({
      request: this.service.deploy(project.id, filename, String(preview['confirmToken']), risks.length > 0, pullImages),
      onStart: () => this.deployBusy.set(true),
      onFinally: () => this.deployBusy.set(false),
      onSuccess: r => {
        const taskID = (r.data as { taskID?: string } | undefined)?.taskID;
        if (taskID) {
          this.tasks.track(String(taskID), '部署 ' + (project.name || project.id), true);
          this.closeEditor();
        } else {
          this.toast.success('部署命令已完成');
        }
      },
      onBizError: r => this.showError(r.msg, '部署'),
      onHttpError: e => this.showError(actionErrorMessage(e, '部署失败'), '部署'),
    });
  }
  newProject() { this.projectName.set(''); this.projectContent.set(defaultCompose); this.createError.set(''); this.createValidated.set(false); this.showCreate.set(true); }
  validateCreate() {
    const n = this.projectName().trim();
    if (!n) { this.setCreateError('请输入项目名称', 'error'); return; }
    this.projectContent.set(this.normalizeCompose(this.projectContent()));
    runAction({
      request: this.service.validate('', 'compose.yaml', this.projectContent()),
      onSuccess: r => {
        const count = Array.isArray((r.data as { services?: unknown[] } | undefined)?.services) ? (r.data as { services: unknown[] }).services.length : 0;
        this.createValidated.set(true);
        this.setCreateError('格式校验通过，包含 ' + count + ' 个服务', 'success');
      },
      onBizError: r => this.setCreateError(this.formatComposeError(r.msg), 'error'),
      onHttpError: e => this.setCreateError(this.formatComposeError(actionErrorMessage(e, '格式校验失败')), 'error'),
    });
  }
  deployNewProject() { this.createProject(true); }
  createProject(deployAfterCreate = false) {
    const n = this.projectName().trim(); if (!n) { this.setCreateError('请输入项目名称', 'error'); return; }
    this.projectContent.set(this.normalizeCompose(this.projectContent()));
    runAction({
      request: this.service.createProject(n, 'compose.yaml', this.projectContent()),
      onStart: () => this.creating.set(true),
      onFinally: () => this.creating.set(false),
      onSuccess: r => {
        if (!deployAfterCreate) { this.closeCreate(); this.toast.success('项目已创建'); return; }
        this.createAndPreview((r.data as { projectId: string }).projectId);
      },
      onBizError: r => this.setCreateError(this.formatComposeError(r.msg || '创建项目失败'), 'error'),
      onHttpError: e => this.setCreateError(this.formatComposeError(actionErrorMessage(e, '创建项目失败')), 'error'),
    });
  }
  private createAndPreview(projectId: string) {
    this.service.projects().subscribe({
      next: projects => {
        const project = projects.data?.projects?.find(item => item.id === projectId);
        if (!project) { this.setCreateError('项目已创建，但读取项目详情失败', 'error'); return; }
        const file = project.files.find(item => item.name === 'compose.yaml') || project.files[0];
        if (!file) { this.setCreateError('项目已创建，但没有找到 Compose 文件', 'error'); return; }
        this.service.deployPreview(project.id, file.name).subscribe({
          next: preview => {
            if (preview.code !== 200) { this.setCreateError(this.formatComposeError(preview.msg || '部署预览失败'), 'error'); return; }
            this.closeCreate();
            this.askDeployment(project, file.name, preview.data);
          },
          error: e => this.setCreateError(this.formatComposeError(e.error?.msg || '部署预览失败'), 'error'),
        });
      },
      error: () => this.setCreateError('项目已创建，但读取项目详情失败', 'error'),
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
  closeCreate(e?: Event) { if (!e || e.target === e.currentTarget) { this.showCreate.set(false); this.creating.set(false); this.createError.set(''); this.createValidated.set(false); this.projectName.set(''); this.projectContent.set(defaultCompose); } }
  private normalizeCompose(input: string): string {
    // 将行首/行内非 ASCII 空白（NBSP、全角空格、零宽字符等）归一化为 ASCII 空格，
    // 避免 yaml.v3 解析报 "could not find expected ':'"。
    if (!input) return input;
    return input.split('\n').map(line => line
      .replace(/[\u00A0\u1680\u2000-\u200A\u202F\u205F\u3000\uFEFF]/g, ' ')
      .replace(/[\u200B\u200C\u200D]/g, '')
    ).join('\n');
  }
  private formatComposeError(detail?: string): string {
    if (!detail) return '未知错误';
    let msg = String(detail);
    const m = msg.match(/YAML 解析失败:\s*(.*)/);
    if (m && m[1]) msg = 'YAML 解析失败：' + m[1].trim();
    const lm = msg.match(/line\s+(\d+)/i);
    if (lm) msg = msg.replace(/line\s+(\d+)/i, '第 $1 行');
    return msg;
  }
  private setCreateError(detail: string, type: 'error' | 'success'): void {
    if (!detail) { this.createError.set(''); this.createErrorType.set(''); return; }
    this.createError.set(detail);
    this.createErrorType.set(type);
  }
  private setEditorError(detail: string): void {
    if (!detail) { this.message.set(''); this.messageType.set(''); return; }
    this.message.set(detail);
    this.messageType.set('error');
  }
  private showError(detail: string, operation = '操作'): void { this.setEditorError(detail); }
  statusLabel(s: string) { return ({ using: '使用中', stopped: '已停止', unused: '未使用', unknown: '未知' } as Record<string, string>)[s] || s; }
  icon(project: ComposeProject) { return this.icons.resolve(project.image || '', this.iconMap()); }
  fallback(event: Event) { (event.target as HTMLImageElement).src = this.icons.actionIcon('images'); }
}
