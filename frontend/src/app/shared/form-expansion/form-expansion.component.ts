import { Component, input } from '@angular/core';
import { MatExpansionModule } from '@angular/material/expansion';

@Component({
  selector: 'dc-form-expansion',
  standalone: true,
  imports: [MatExpansionModule],
  templateUrl: './form-expansion.component.html',
  styleUrl: './form-expansion.component.scss',
})
export class FormExpansionComponent {
  readonly title = input.required<string>();
  readonly description = input('');
  readonly expanded = input(false);
  readonly disabled = input(false);
}
