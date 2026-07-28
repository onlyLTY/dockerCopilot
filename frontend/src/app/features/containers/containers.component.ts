import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ContainerService, ContainerRow } from '../../core/container.service';
import { IconService, IconMap } from '../../core/icon.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-containers',
  standalone: true,
  imports: [CommonModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './containers.component.html',
})
export class ContainersComponent {
  private readonly service = inject(ContainerService); private readonly icons = inject(IconService);
  containers: ContainerRow[] = []; iconMap: IconMap = {}; error = ''; loading = false;
  get runningCount() { return this.containers.filter(x => this.isRunning(x)).length; } get updateCount() { return this.containers.filter(x => x.haveUpdate).length; }
  constructor() { this.load(); this.icons.list().subscribe({ next: r => this.iconMap = r.data || {} }); }
  load(): void { this.loading = true; this.error = ''; this.service.list().subscribe({ next: r => { this.loading = false; if (r.code === 200 || r.code === 0) this.containers = r.data || []; else this.error = r.msg || '读取容器失败'; }, error: e => { this.loading = false; this.error = e.error?.msg || '读取容器失败'; } }); }
  isRunning(x: ContainerRow) { return x.status.includes('Up') || x.status.includes('running') || x.status.includes('运行'); }
  icon(x: ContainerRow) { return this.icons.resolve(x.usingImage || x.name, this.iconMap); } fallback(e: Event) { (e.target as HTMLImageElement).src = this.icons.actionIcon('containers'); }
  action(x: ContainerRow, fn: (id: string) => any) { fn(x.id).subscribe({ next: (r: any) => r.code === 200 ? this.load() : this.error = r.msg || '操作失败', error: (e: any) => this.error = e.error?.msg || '操作失败' }); }
  start(x: ContainerRow) { this.action(x, id => this.service.start(id)); } stop(x: ContainerRow) { this.action(x, id => this.service.stop(id)); } restart(x: ContainerRow) { this.action(x, id => this.service.restart(id)); } update(x: ContainerRow) { this.action(x, id => this.service.update(id)); }
}
