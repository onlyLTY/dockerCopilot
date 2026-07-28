import { Component, inject, signal } from '@angular/core';
import { VersionService } from '../../core/version.service';

@Component({ selector: 'dc-about', standalone: true, templateUrl: './about.component.html' })
export class AboutComponent {
  private readonly service = inject(VersionService); readonly version = signal({ version: '', buildDate: '' });
  constructor() { this.service.local().subscribe({ next: result => { if (result.code === 200) this.version.set(result.data); } }); }
}
