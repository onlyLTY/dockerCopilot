import { Component, Input, input, output } from '@angular/core';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'dc-modal-heading',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './modal-heading.component.html',
  styleUrl: './modal-heading.component.scss',
})
export class ModalHeadingComponent {
  readonly title = input.required<string>();
  readonly subtitle = input('');
  @Input() subtitleClass = '';
  readonly closed = output<void>();
}
