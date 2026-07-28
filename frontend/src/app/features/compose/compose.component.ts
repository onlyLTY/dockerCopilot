import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ComposeService, ComposeProject, ProjectsData } from '../../core/compose.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

const defaultCompose = `services:\n  app:\n    image: nginx:alpine\n    ports:\n      - "8080:80"\n`;

@Component({
  selector: 'dc-compose',
  standalone: true,
  imports: [CommonModule, FormsModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './compose.component.html',
})
export class ComposeComponent {
  private readonly service = inject(ComposeService); data?: ProjectsData; editorProject?: ComposeProject; filename = ''; content = ''; version = ''; message = ''; messageType = ''; preview?: Record<string, any>; risks: any[] = []; showCreate = false; creating = false; loading = false; projectName = ''; projectContent = defaultCompose; createError = '';
  constructor() { this.load(); }
  load() { this.loading = true; this.message = ''; this.messageType = ''; this.service.projects().subscribe({ next: r => { this.loading = false; if (r.code === 200) this.data = r.data; else this.showError(r.msg); }, error: e => { this.loading = false; this.showError(e.error?.msg || '读取项目失败'); } }); }
  openEditor(p: ComposeProject) { this.editorProject = p; this.preview = undefined; this.message = ''; this.messageType = ''; const f = p.files.find(x => x.name === 'compose.yaml') || p.files[0]; if (f) this.openFile(f.name); }
  openFile(n: string) { if (!this.editorProject) return; this.filename = n; this.service.file(this.editorProject.id, n).subscribe({ next: r => r.code === 200 ? (this.content = r.data.content, this.version = r.data.version, this.message = '') : this.showError(r.msg), error: e => this.showError(e.error?.msg || '读取文件失败') }); }
  save() { if (!this.editorProject) return; this.service.update(this.editorProject.id, this.filename, this.content, this.version).subscribe({ next: r => r.code === 200 ? (this.version = r.data.version, this.message = '已保存') : this.showError(r.msg), error: e => this.showError(e.error?.msg || '保存失败') }); }
  validate() { if (!this.editorProject) return; this.service.validate(this.editorProject.id, this.filename, this.content).subscribe({ next: r => r.code === 200 ? (this.message = '校验通过，包含 ' + r.data.services.length + ' 个服务', this.messageType = '') : this.showError(r.msg), error: e => this.showError(e.error?.msg || '校验失败') }); }
  previewDeploy() { if (!this.editorProject) return; this.service.deployPreview(this.editorProject.id, this.filename).subscribe({ next: r => r.code === 200 ? (this.preview = r.data, this.risks = (r.data['risks'] as any[]) || [], this.message = '预览已生成') : this.showError(r.msg), error: e => this.showError(e.error?.msg || '预览失败') }); }
  deploy() { if (!this.editorProject || !this.preview) return; this.service.deploy(this.editorProject.id, this.filename, String(this.preview['confirmToken']), true).subscribe({ next: r => { if (r.code === 200) { this.message = '部署命令已完成'; this.load(); } else this.showError(r.msg); }, error: e => this.showError(e.error?.msg || '部署失败') }); }
  newProject() { this.projectName = ''; this.projectContent = defaultCompose; this.createError = ''; this.showCreate = true; }
  createProject() { const n = this.projectName.trim(); if (!n) { this.createError = '请输入项目名称'; return; } this.creating = true; this.service.createProject(n, 'compose.yaml', this.projectContent).subscribe({ next: r => { this.creating = false; if (r.code !== 200) { this.createError = r.msg; return; } this.closeCreate(); this.load(); }, error: e => { this.creating = false; this.createError = e.error?.msg || '创建项目失败'; } }); }
  closeEditor(e?: Event) { if (!e || e.target === e.currentTarget) this.editorProject = undefined; }
  closeCreate(e?: Event) { if (!e || e.target === e.currentTarget) { this.showCreate = false; this.creating = false; this.createError = ''; this.projectName = ''; this.projectContent = defaultCompose; } }
  showError(m: string) { this.message = m; this.messageType = 'error'; }
  statusLabel(s: string) { return ({ using: '使用中', stopped: '已停止', unused: '未使用', unknown: '未知' } as Record<string, string>)[s] || s; }
}
