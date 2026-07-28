import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { ImageRow, ImageService } from '../../core/image.service';
import { IconService, IconMap } from '../../core/icon.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-images',
  standalone: true,
  imports: [CommonModule, PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './images.component.html',
})
export class ImagesComponent {
  private readonly service = inject(ImageService); private readonly icons = inject(IconService); images: ImageRow[] = []; iconMap: IconMap = {}; error = ''; loading = false; cleaning = false;
  get usedCount() { return this.images.filter(x => x.inUsed).length; } get untaggedCount() { return this.images.filter(x => this.isUntagged(x)).length; } get unusedCount() { return this.images.filter(x => !x.inUsed).length; }
  constructor() { this.load(); this.icons.list().subscribe({ next: r => this.iconMap = r.data || {} }); }
  load() { this.loading = true; this.error = ''; this.service.list().subscribe({ next: r => { this.loading = false; if (r.code === 200) this.images = r.data || []; else this.error = r.msg || '读取镜像失败'; }, error: e => { this.loading = false; this.error = e.error?.msg || '读取镜像失败'; } }); }
  isUntagged(x: ImageRow) { return !x.tag || ['<none>', 'none'].includes(x.tag.toLowerCase()); }
  cleanup(kind: 'untagged' | 'unused') { const count = kind === 'untagged' ? this.untaggedCount : this.unusedCount; const label = kind === 'untagged' ? '无 Tag' : '未使用'; if (!count || !confirm(`确定清理 ${count} 个${label}镜像吗？`)) return; this.cleaning = true; this.error = ''; this.service.cleanup(kind).subscribe({ next: r => { this.cleaning = false; if (r.code === 200) this.load(); else this.error = r.msg || '清理镜像失败'; }, error: e => { this.cleaning = false; this.error = e.error?.msg || '清理镜像失败'; } }); }
  icon(x: ImageRow) { return this.icons.resolve(x.name, this.iconMap); } fallback(e: Event) { (e.target as HTMLImageElement).src = this.icons.actionIcon('images'); }
  remove(x: ImageRow, force: boolean) { if (!confirm((force ? '强制删除' : '删除') + '镜像 ' + x.name + ':' + x.tag + '？')) return; this.service.remove(x.id, force).subscribe({ next: r => r.code === 200 ? this.load() : this.error = r.msg || '删除失败', error: e => this.error = e.error?.msg || '删除失败' }); }
}
