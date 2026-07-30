import { Component, input } from '@angular/core';

@Component({
  selector: 'dc-page-heading',
  standalone: true,
  templateUrl: './page-heading.component.html',
  styleUrl: './page-heading.component.scss',
})
export class PageHeadingComponent {
  readonly title = input.required<string>();
  readonly subtitle = input('');
}
