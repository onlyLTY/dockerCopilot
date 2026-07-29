import { Component, inject, signal, computed } from '@angular/core';
import { forkJoin, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { FormsModule } from '@angular/forms';
import { IconService, IconMap } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';

@Component({
  selector: 'dc-icons',
  standalone: true,
  imports: [FormsModule, IconComponent],
  templateUrl: './icons.component.html',
})
export class IconsComponent {
  private readonly service = inject(IconService); private readonly toast = inject(ToastService); private readonly confirm = inject(ConfirmService);
  // 图标 map 来自服务常驻缓存
  readonly icons = computed<IconMap>(() => this.service.cache.data() || {});
  query = ''; readonly show = signal<boolean>(false); readonly editing = signal<string>(''); name = ''; readonly file = signal<File | undefined>(undefined); readonly uploading = signal<boolean>(false); readonly error = signal<string>('');
  readonly filtered = computed<[string, string][]>(() => { const q = this.query.trim().toLowerCase(); return Object.entries(this.icons()).filter(item => !q || item[0].toLowerCase().includes(q)); });
  readonly selectionMode = signal(false); readonly selected = signal<Set<string>>(new Set()); readonly busy = signal(false);
  enterSelection() { this.selectionMode.set(true); }
  exitSelection() { this.selectionMode.set(false); this.selected.set(new Set()); }
  isSelected(name: string) { return this.selected().has(name); }
  toggleSelect(name: string) { const next = new Set(this.selected()); next.has(name) ? next.delete(name) : next.add(name); this.selected.set(next); }
  readonly allSelected = computed(() => this.filtered().length > 0 && this.filtered().every(item => this.selected().has(item[0])));
  toggleAll() { const next = new Set(this.selected()); if (this.allSelected()) this.filtered().forEach(item => next.delete(item[0])); else this.filtered().forEach(item => next.add(item[0])); this.selected.set(next); }
  bulkRemove() {
    const names = this.filtered().map(item => item[0]).filter(name => this.selected().has(name));
    if (!names.length || this.busy()) return;
    this.confirm.open({ title: '批量删除图标', message: `确定删除选中的 ${names.length} 个图标吗？`, confirmText: '批量删除', danger: true }).then(ok => { if (ok) this.runBulkRemove(names); });
  }
  private runBulkRemove(names: string[]) {
    this.busy.set(true);
    forkJoin(names.map(name => this.service.remove(name).pipe(
      map(r => ({ name, ok: r.code === 200, msg: r.msg })),
      catchError(e => of({ name, ok: false, msg: e.error?.msg || e.message || '请求错误' })),
    ))).subscribe(results => {
      this.busy.set(false);
      const fail = results.filter(r => !r.ok).length;
      if (fail) this.toast.error(`批量删除完成 ${results.length - fail} 个，失败 ${fail} 个`);
      else this.toast.success(`已删除 ${names.length} 个图标`);
      this.exitSelection();
    });
  }
  constructor() { this.service.ensureLoaded(); }
  openUpload(): void { this.editing.set(''); this.name = ''; this.file.set(undefined); this.error.set(''); this.show.set(true); }
  edit(name: string): void { this.editing.set(name); this.name = name; this.file.set(undefined); this.error.set(''); this.show.set(true); }
  choose(event: Event): void { const input = event.target as HTMLInputElement; this.file.set(input.files?.[0]); }
  upload(): void { const file = this.file(); if (!file || !this.name.trim()) return; this.uploading.set(true); this.service.upload(this.name.trim(), file).subscribe({ next: result => { this.uploading.set(false); if (result.code === 200) { this.show.set(false); this.toast.success('图标已保存'); } else this.error.set(result.msg); }, error: error => { this.uploading.set(false); this.error.set(error.error?.msg || '上传失败'); } }); }
  async remove(name: string): Promise<void> { if (!(await this.confirm.open({ title: '删除图标', message: `删除图标 ${name}？`, confirmText: '删除', danger: true }))) return; this.service.remove(name).subscribe({ next: result => { if (result.code === 200) this.toast.success('图标已删除'); else this.toast.error(`删除失败：${result.msg || '未知错误'}`); }, error: e => this.toast.error(`删除失败：${e.error?.msg || e.message || '请求错误'}`) }); }
  close(event: Event): void { if (event.target === event.currentTarget) this.show.set(false); }
  fallback(event: Event): void { (event.target as HTMLImageElement).src = 'assets/imageIcons/defaultIcon.png'; }
}
