import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';

@Component({
  selector: 'dc-section-toolbar',
  standalone: true,
  imports: [CommonModule],
  template: `<div class="section-toolbar"><div><h2>{{ title }}</h2><p *ngIf="subtitle">{{ subtitle }}</p></div><div class="section-toolbar-actions"><ng-content /></div></div>`,
})
export class SectionToolbarComponent {
  @Input() title = '';
  @Input() subtitle = '';
}
