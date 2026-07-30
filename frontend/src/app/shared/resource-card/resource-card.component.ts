import { Component, input } from '@angular/core';

@Component({
  selector: 'dc-resource-card',
  standalone: true,
  templateUrl: './resource-card.component.html',
  styleUrl: './resource-card.component.scss',
})
export class ResourceCardComponent {
  readonly selected = input(false);
}
