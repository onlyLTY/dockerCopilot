import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { IconService, IconMap } from '../../core/icon.service';
import { IconComponent } from '../../shared/icon.component';

@Component({
  selector: 'dc-icons',
  standalone: true,
  imports: [CommonModule, FormsModule, IconComponent],
  templateUrl: './icons.component.html',
})
export class IconsComponent {
  private readonly service = inject(IconService); icons: IconMap = {}; query = ''; show = false; editing = ''; name = ''; file?: File; uploading = false; error = '';
  get filtered(): [string, string][] { const q = this.query.trim().toLowerCase(); return Object.entries(this.icons).filter(item => !q || item[0].toLowerCase().includes(q)); }
  constructor() { this.load(); }
  load(): void { this.service.list().subscribe({ next: result => this.icons = result.data || {} }); }
  openUpload(): void { this.editing = ''; this.name = ''; this.file = undefined; this.error = ''; this.show = true; }
  edit(name: string): void { this.editing = name; this.name = name; this.file = undefined; this.error = ''; this.show = true; }
  choose(event: Event): void { const input = event.target as HTMLInputElement; this.file = input.files?.[0]; }
  upload(): void { if (!this.file || !this.name.trim()) return; this.uploading = true; this.service.upload(this.name.trim(), this.file).subscribe({ next: result => { this.uploading = false; if (result.code === 200) { this.show = false; this.load(); } else this.error = result.msg; }, error: error => { this.uploading = false; this.error = error.error?.msg || '上传失败'; } }); }
  remove(name: string): void { if (!confirm('删除图标 ' + name + '？')) return; this.service.remove(name).subscribe({ next: result => { if (result.code === 200) this.load(); } }); }
  close(event: Event): void { if (event.target === event.currentTarget) this.show = false; }
  fallback(event: Event): void { (event.target as HTMLImageElement).src = 'assets/imageIcons/defaultIcon.png'; }
}
