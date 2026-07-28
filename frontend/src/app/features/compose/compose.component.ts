import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ComposeService, ComposeProject, ProjectsData } from '../../core/compose.service';
import { ToastService } from '../../core/toast.service';
import { TaskService } from '../../core/task.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

const defaultCompose = `services:\n  app:\n    image: nginx:alpine\n    ports:\n      - "8080:80"\n`;

@Component({
  selector: 'dc-compose',
  standalone: true,
  imports: [FormsModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './compose.component.html',
})
export class ComposeComponent {
  private readonly service = inject(ComposeService); private readonly toast = inject(ToastService); private readonly tasks = inject(TaskService);
  // 项目数据、加载态来自服务常驻缓存
  readonly data = computed(() => this.service.cache.data());
  readonly loading = this.service.cache.loading;
  readonly editorProject = signal<ComposeProject | undefined>(undefined); readonly filename = signal(''); content = ''; readonly version = signal(''); readonly message = signal(''); readonly messageType = signal(''); readonly preview = signal<Record<string, any> | undefined>(undefined); readonly risks = signal<any[]>([]); readonly showCreate = signal(false); readonly creating = signal(false); projectName = ''; projectContent = defaultCompose; readonly createError = signal('');
  constructor() { this.service.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  openEditor(p: ComposeProject) { this.editorProject.set(p); this.preview.set(undefined); this.message.set(''); this.messageType.set(''); const f = p.files.find(x => x.name === 'compose.yaml') || p.files[0]; if (f) this.openFile(f.name); }
  openFile(n: string) { const ep = this.editorProject(); if (!ep) return; this.filename.set(n); this.service.file(ep.id, n).subscribe({ next: r => r.code === 200 ? (this.content = r.data.content, this.version.set(r.data.version), this.message.set('')) : this.showError(r.msg), error: e => this.showError(e.error?.msg || '读取文件失败') }); }
  save() { const ep = this.editorProject(); if (!ep) return; this.service.update(ep.id, this.filename(), this.content, this.version()).subscribe({ next: r => r.code === 200 ? (this.version.set(r.data.version), this.message.set('已保存')) : this.showError(r.msg), error: e => this.showError(e.error?.msg || '保存失败') }); }
  validate() { const ep = this.editorProject(); if (!ep) return; this.service.validate(ep.id, this.filename(), this.content).subscribe({ next: r => r.code === 200 ? (this.message.set('校验通过，包含 ' + r.data.services.length + ' 个服务'), this.messageType.set('')) : this.showError(r.msg), error: e => this.showError(e.error?.msg || '校验失败') }); }
  previewDeploy() { const ep = this.editorProject(); if (!ep) return; this.service.deployPreview(ep.id, this.filename()).subscribe({ next: r => r.code === 200 ? (this.preview.set(r.data), this.risks.set((r.data['risks'] as any[]) || []), this.message.set('预览已生成')) : this.showError(r.msg), error: e => this.showError(e.error?.msg || '预览失败') }); }
  deploy() {
    const ep = this.editorProject(); const pv = this.preview(); if (!ep || !pv) return;
    this.service.deploy(ep.id, this.filename(), String(pv['confirmToken']), true).subscribe({
      next: r => {
        if (r.code === 200) {
          const taskID = (r.data as any)?.['taskID'];
          if (taskID) { this.tasks.track(String(taskID), '部署 ' + (ep.name || ep.id), true); this.closeEditor(); }
          else { this.message.set('部署命令已完成'); this.toast.success('部署命令已完成'); }
        } else this.showError(r.msg);
      },
      error: e => this.showError(e.error?.msg || '部署失败'),
    });
  }
  newProject() { this.projectName = ''; this.projectContent = defaultCompose; this.createError.set(''); this.showCreate.set(true); }
  createProject() { const n = this.projectName.trim(); if (!n) { this.createError.set('请输入项目名称'); return; } this.creating.set(true); this.service.createProject(n, 'compose.yaml', this.projectContent).subscribe({ next: r => { this.creating.set(false); if (r.code !== 200) { this.createError.set(r.msg); return; } this.closeCreate(); this.toast.success('项目已创建'); }, error: e => { this.creating.set(false); this.createError.set(e.error?.msg || '创建项目失败'); } }); }
  closeEditor(e?: Event) { if (!e || e.target === e.currentTarget) this.editorProject.set(undefined); }
  closeCreate(e?: Event) { if (!e || e.target === e.currentTarget) { this.showCreate.set(false); this.creating.set(false); this.createError.set(''); this.projectName = ''; this.projectContent = defaultCompose; } }
  showError(m: string) { this.message.set(m); this.messageType.set('error'); this.toast.error(m); }
  statusLabel(s: string) { return ({ using: '使用中', stopped: '已停止', unused: '未使用', unknown: '未知' } as Record<string, string>)[s] || s; }
}
