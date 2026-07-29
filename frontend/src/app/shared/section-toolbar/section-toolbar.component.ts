import { Component, input } from '@angular/core';

@Component({
  selector: 'dc-section-toolbar',
  standalone: true,
  template: `<div class="section-toolbar"><div><h2>{{ title() }}</h2>@if (subtitle()) {<p>{{ subtitle() }}</p>}</div><div class="section-toolbar-actions d-flex items-center gap-8"><ng-content /></div></div>`,
})
export class SectionToolbarComponent {
  readonly title = input('');
  readonly subtitle = input('');
}
