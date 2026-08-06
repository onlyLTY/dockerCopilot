import { Component, input } from '@angular/core';

@Component({
  selector: 'dc-section-toolbar',
  standalone: true,
  template: `<div class="section-toolbar d-flex items-center justify-between gap-16 mb-16 min-h-42">
    <div>
      <h2 class="m-0 text-black fs-18">{{ title() }}</h2>
      @if (subtitle()) {
        <p class="m-0 mt-4 fs-12 text-muted">{{ subtitle() }}</p>
      }
    </div>
    <div class="section-toolbar-actions d-flex items-center gap-8"><ng-content /></div>
  </div>`,
  styleUrl: './section-toolbar.component.scss',
})
export class SectionToolbarComponent {
  readonly title = input('');
  readonly subtitle = input('');
}
