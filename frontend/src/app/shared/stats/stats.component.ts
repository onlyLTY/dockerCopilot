import { Component, input } from '@angular/core';

export type StatTone = 'green' | 'amber' | 'violet' | 'red' | 'blue';

export interface StatItem {
  value: string | number;
  label: string;
  tone?: StatTone;
}

@Component({
  selector: 'dc-stats',
  standalone: true,
  templateUrl: './stats.component.html',
  styleUrl: './stats.component.scss',
})
export class StatsComponent {
  readonly items = input.required<readonly StatItem[]>();
  readonly columns = input.required<2 | 3 | 4>();
}
