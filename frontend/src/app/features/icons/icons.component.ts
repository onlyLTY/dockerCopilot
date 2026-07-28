import { Component, inject, signal, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { IconService, IconMap } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-icons',
  standalone: true,
  imports: [FormsModule, IconComponent],
  templateUrl: './icons.component.html',
})
export class IconsComponent {
  private readonly service = inject(IconService); private readonly toast = inject(ToastService);
  // 图标 map 来自服务常驻缓存
  readonly icons = computed<IconMap>(() => this.service.cache.data() || {});
  query = ''; readonly show = signal<boolean>(false); readonly editing = signal<string>(''); name = ''; readonly file = signal<File | undefined>(undefined); readonly uploading = signal<boolean>(false); readonly error = signal<string>('');
  readonly filtered = computed<[string, string][]>(() => { const q = this.query.trim().toLowerCase(); return Object.entries(this.icons()).filter(item => !q || item[0].toLowerCase().includes(q)); });
  constructor() { this.service.ensureLoaded(); }
  openUpload(): void { this.editing.set(''); this.name = ''; this.file.set(undefined); this.error.set(''); this.show.set(true); }
  edit(name: string): void { this.editing.set(name); this.name = name; this.file.set(undefined); this.error.set(''); this.show.set(true); }
  choose(event: Event): void { const input = event.target as HTMLInputElement; this.file.set(input.files?.[0]); }
  upload(): void { const file = this.file(); if (!file || !this.name.trim()) return; this.uploading.set(true); this.service.upload(this.name.trim(), file).subscribe({ next: result => { this.uploading.set(false); if (result.code === 200) { this.show.set(false); this.toast.success('图标已保存'); } else this.error.set(result.msg); }, error: error => { this.uploading.set(false); this.error.set(error.error?.msg || '上传失败'); } }); }
  remove(name: string): void { if (!confirm('删除图标 ' + name + '？')) return; this.service.remove(name).subscribe({ next: result => { if (result.code === 200) this.toast.success('图标已删除'); else this.toast.error(`删除失败：${result.msg || '未知错误'}`); }, error: e => this.toast.error(`删除失败：${e.error?.msg || e.message || '请求错误'}`) }); }
  close(event: Event): void { if (event.target === event.currentTarget) this.show.set(false); }
  fallback(event: Event): void { (event.target as HTMLImageElement).src = 'assets/imageIcons/defaultIcon.png'; }
}
