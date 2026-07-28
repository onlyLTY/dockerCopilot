import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { VersionService } from '../../core/version.service';

@Component({ selector: 'dc-about', standalone: true, imports: [CommonModule], templateUrl: './about.component.html' })
export class AboutComponent {
  private readonly service = inject(VersionService); version = { version: '', buildDate: '' };
  constructor() { this.service.local().subscribe({ next: result => { if (result.code === 200) this.version = result.data; } }); }
}
