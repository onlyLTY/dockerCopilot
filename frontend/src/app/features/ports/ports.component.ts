import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { PortService } from '../../core/port.service';
import { PortUsage } from '../../core/compose.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-ports',
  standalone: true,
  imports: [CommonModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './ports.component.html',
})
export class PortsComponent {
  private readonly service = inject(PortService);
  data = { ports: [] as PortUsage[], conflicts: [] as string[], warnings: [] as string[] };
  error = ''; loading = false;
  constructor() { this.load(); }
  load() { this.loading = true; this.error = ''; this.service.list().subscribe({ next: r => { this.loading = false; if (r.code === 200) this.data = r.data; else this.error = r.msg || '读取端口失败'; }, error: e => { this.loading = false; this.error = e.error?.msg || '读取端口失败'; } }); }
}
