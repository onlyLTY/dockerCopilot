import { Component, inject, signal, computed } from '@angular/core';
import { ImageRow, ImageService } from '../../core/image.service';
import { IconService } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { PageStateComponent } from '../../shared/page-state.component';
import { SectionToolbarComponent } from '../../shared/section-toolbar.component';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-images',
  standalone: true,
  imports: [PageStateComponent, SectionToolbarComponent, IconComponent],
  templateUrl: './images.component.html',
})
export class ImagesComponent {
  private readonly service = inject(ImageService); private readonly icons = inject(IconService); private readonly toast = inject(ToastService);
  // 数据、加载态、错误态来自服务常驻缓存
  readonly images = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly cleaning = signal<boolean>(false);
  readonly usedCount = computed(() => this.images().filter(x => x.inUsed).length); readonly untaggedCount = computed(() => this.images().filter(x => this.isUntagged(x)).length); readonly unusedCount = computed(() => this.images().filter(x => !x.inUsed).length);
  constructor() { this.service.ensureLoaded(); this.icons.ensureLoaded(); }
  refresh() { this.service.refresh(); }
  isUntagged(x: ImageRow) { return !x.tag || ['<none>', 'none'].includes(x.tag.toLowerCase()); }
  cleanup(kind: 'untagged' | 'unused') { const count = kind === 'untagged' ? this.untaggedCount() : this.unusedCount(); const label = kind === 'untagged' ? '无 Tag' : '未使用'; if (!count || !confirm(`确定清理 ${count} 个${label}镜像吗？`)) return; this.cleaning.set(true); this.service.cleanup(kind).subscribe({ next: r => { this.cleaning.set(false); if (r.code === 200) { this.toast.success(`已清理 ${r.data?.deleted ?? ''} 个${label}镜像`); } else this.toast.error(`清理失败：${r.msg || '未知错误'}`); }, error: e => { this.cleaning.set(false); this.toast.error(`清理失败：${e.error?.msg || e.message || '请求错误'}`); } }); }
  icon(x: ImageRow) { return this.icons.resolve(x.name, this.iconMap()); } fallback(e: Event) { (e.target as HTMLImageElement).src = this.icons.actionIcon('images'); }
  remove(x: ImageRow, force: boolean) { const label = force ? '强制删除' : '删除'; if (!confirm(label + '镜像 ' + x.name + ':' + x.tag + '？')) return; this.service.remove(x.id, force).subscribe({ next: r => { if (r.code === 200) { this.toast.success(`${x.name} ${label}成功`); } else this.toast.error(`${x.name} ${label}失败：${r.msg || '未知错误'}`); }, error: e => this.toast.error(`${x.name} ${label}失败：${e.error?.msg || e.message || '请求错误'}`) }); }
}
