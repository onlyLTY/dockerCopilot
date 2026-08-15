import { Component, inject, output, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { switchMap } from 'rxjs';
import { ApiResponse } from '../../../../core/compose.service';
import { ModalHeadingComponent } from '../../../../shared/modal-heading/modal-heading.component';
import { StatsComponent, StatItem } from '../../../../shared/stats/stats.component';

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

@Component({
  selector: 'dc-log-dialog',
  standalone: true,
  imports: [ModalHeadingComponent, StatsComponent],
  templateUrl: './log-dialog.component.html',
  styleUrl: './log-dialog.component.scss',
})
export class LogDialogComponent {
  private readonly http = inject(HttpClient);
  private requestID = 0;

  readonly closed = output<void>();
  readonly loading = signal(false);
  readonly error = signal('');
  readonly logs = signal<LogEntry[]>([]);
  readonly level = signal('error');
  readonly stats = signal<StatItem[]>([]);

  constructor() {
    this.fetch('error');
  }

  selectLevel(level: string): void {
    if (!level || level === this.level()) return;
    this.level.set(level);
    this.fetch(level);
  }

  close(): void {
    this.requestID++;
    this.closed.emit();
  }

  private fetch(level: string): void {
    const requestID = ++this.requestID;
    this.loading.set(true);
    this.error.set('');
    const params = new URLSearchParams({ limit: '100', level });
    this.http.get<ApiResponse<{ entries: LogEntry[] }>>('/api/logs?' + params.toString()).subscribe({
      next: response => {
        if (requestID !== this.requestID) return;
        this.loading.set(false);
        if (response.code === 200) {
          const entries = response.data?.entries || [];
          this.logs.set(entries);
          this.stats.set(this.createStats(level, entries.length));
        } else {
          this.error.set(response.msg || '读取日志失败');
        }
      },
      error: responseError => {
        if (requestID !== this.requestID) return;
        this.loading.set(false);
        this.error.set(responseError.error?.msg || '读取日志失败');
      },
    });
  }

  private createStats(level: string, count: number): StatItem[] {
    const value = (key: string) => (level === key ? count : '-');
    return [
      { key: 'error', value: value('error'), label: 'ERROR', tone: 'red' },
      { key: 'warn', value: value('warn'), label: 'WARN', tone: 'amber' },
      { key: 'info', value: value('info'), label: 'INFO', tone: 'blue' },
      { key: 'debug', value: value('debug'), label: 'DEBUG' },
    ];
  }
}
