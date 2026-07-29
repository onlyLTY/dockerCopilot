import { Component, inject, computed } from '@angular/core';
import { PortService } from '../../core/port.service';
import { PortUsage } from '../../core/compose.service';
import { IconService } from '../../core/icon.service';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';

interface PortLine { hostPort: string; containerPort: string; protocol: string; }
interface PortGroup { containerName: string; project: string; image: string; ports: PortLine[]; }

@Component({
  selector: 'dc-ports',
  standalone: true,
  imports: [PageStateComponent, IconComponent],
  templateUrl: './ports.component.html',
})
export class PortsComponent {
  private readonly service = inject(PortService); private readonly icons = inject(IconService);
  // 数据、加载态、错误态来自服务常驻缓存
  readonly data = computed(() => this.service.cache.data() || { ports: [] as PortUsage[], conflicts: [] as string[], warnings: [] as string[] });
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  // 按容器聚合：一个容器一张卡片，仅展示端口
  readonly groups = computed<PortGroup[]>(() => {
    const map = new Map<string, PortGroup>();
    for (const p of this.data().ports) {
      const key = p.containerID || p.containerName;
      if (!map.has(key)) map.set(key, { containerName: p.containerName, project: p.project, image: p.image || '', ports: [] });
      const g = map.get(key)!;
      const line: PortLine = { hostPort: p.hostPort, containerPort: p.containerPort, protocol: p.protocol };
      const label = `${line.hostPort}|${line.containerPort}|${line.protocol}`;
      if (!g.ports.some(x => `${x.hostPort}|${x.containerPort}|${x.protocol}` === label)) g.ports.push(line);
    }
    return Array.from(map.values());
  });
  constructor() { this.service.ensureLoaded(); this.icons.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  icon(g: PortGroup) { return this.icons.resolve(g.image, this.iconMap()); }
  fallback(e: Event) { (e.target as HTMLImageElement).src = this.icons.actionIcon('containers'); }
}
