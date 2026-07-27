import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ContainerService, ContainerRow } from '../../core/container.service';
import { ComposeService, ComposeProject, ProjectsData } from '../../core/compose.service';

@Component({ selector: 'dc-dashboard', standalone: true, imports: [CommonModule], template: `
<section class="page-heading"><div><p class="eyebrow">DOCKER OPERATIONS</p><h1>容器管理</h1><p class="subtitle">管理您的 Docker 容器、Compose 项目和端口使用情况</p></div><button class="primary" (click)="load()">↻ 刷新</button></section>
<section class="stats" *ngIf="data as d"><div><strong>{{ d.summary.total }}</strong><span>Compose 项目</span></div><div class="green"><strong>{{ d.summary.using }}</strong><span>使用中</span></div><div class="amber"><strong>{{ d.summary.stopped }}</strong><span>已停止</span></div><div class="violet"><strong>{{ d.summary.unused }}</strong><span>未使用</span></div><div class="red"><strong>{{ d.summary.unknown }}</strong><span>无法判断</span></div></section>
<section class="panel"><div class="panel-head"><div><h2>Compose 项目</h2><p>按 Docker Compose 标签匹配运行状态</p></div><div class="filters"><button [class.selected]="filter === 'all'" (click)="filter = 'all'">全部</button><button [class.selected]="filter === 'using'" (click)="filter = 'using'">使用中</button><button [class.selected]="filter === 'unused'" (click)="filter = 'unused'">未使用</button></div></div>
<div class="table-wrap"><table><thead><tr><th>项目</th><th>目录</th><th>容器</th><th>端口</th><th>状态</th></tr></thead><tbody><tr *ngFor="let project of filtered"><td><b>{{ project.name || '未解析项目' }}</b><small>{{ project.id }}</small></td><td class="muted">{{ project.root }}</td><td>{{ project.containers.length }}</td><td>{{ project.ports.length }}</td><td><span class="status" [class]="project.status">{{ statusLabel(project.status) }}</span></td></tr></tbody></table><div class="empty" *ngIf="!filtered.length">暂无 Compose 项目</div></div></section>
<section class="panel container-panel"><div class="panel-head"><div><h2>容器概览</h2><p>现有容器的运行状态</p></div></div><div class="container-grid"><article *ngFor="let container of containers"><div class="container-icon">□</div><div><b>{{ container.name }}</b><span>{{ container.usingImage }}</span><small>{{ container.status }}</small></div></article></div></section>
`, })
export class DashboardComponent {
  private readonly compose = inject(ComposeService); private readonly containerService = inject(ContainerService);
  data?: ProjectsData; containers: ContainerRow[] = []; filter = 'all';
  get filtered(): ComposeProject[] { return (this.data?.projects ?? []).filter(item => this.filter === 'all' || item.status === this.filter); }
  constructor() { this.load(); }
  load(): void { this.compose.projects().subscribe({ next: result => this.data = result.data, error: () => this.data = { summary: { total: 0, using: 0, stopped: 0, unused: 0, unknown: 0 }, projects: [] } }); this.containerService.list().subscribe({ next: result => this.containers = result.data ?? [] }); }
  statusLabel(status: string): string { return ({ using: '使用中', stopped: '已停止', unused: '未使用', unknown: '无法判断' } as Record<string, string>)[status] ?? status; }
}
