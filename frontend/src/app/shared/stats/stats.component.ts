import { Component, input, output } from '@angular/core';

export type StatTone = 'green' | 'amber' | 'violet' | 'red' | 'blue';

export interface StatItem {
  value: string | number;
  label: string;
  tone?: StatTone;
  key?: string;
}

@Component({
  selector: 'dc-stats',
  standalone: true,
  templateUrl: './stats.component.html',
  styleUrl: './stats.component.scss',
})
export class StatsComponent {
  readonly items = input.required<readonly StatItem[]>();
  readonly columns = input.required<2 | 3 | 4 | 5>();
  readonly activeKey = input('');
  readonly selectedChange = output<string>();

  select(item: StatItem): void {
    if (item.key) this.selectedChange.emit(this.activeKey() === item.key ? '' : item.key);
  }
}
