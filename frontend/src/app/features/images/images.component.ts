import { Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { ImageRow, ImageService } from '../../core/image.service';
import { IconService } from '../../core/icon.service';
import { ToastService } from '../../core/toast.service';
import { SoftRefreshHandle, startSoftRefresh } from '../../core/soft-refresh';
import { actionErrorMessage, actionLabel, runAction } from '../../core/run-action';
import { PageStateComponent } from '../../shared/page-state/page-state.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { ConfirmService } from '../../core/confirm.service';
import { ResourceCardComponent } from '../../shared/resource-card/resource-card.component';
import { StatsComponent, StatItem } from '../../shared/stats/stats.component';
import { PageHeadingComponent } from '../../shared/page-heading/page-heading.component';

@Component({
  selector: 'dc-images',
  standalone: true,
  imports: [
    PageStateComponent,
    IconComponent,
    ResourceCardComponent,
    StatsComponent,
    PageHeadingComponent,
  ],
  templateUrl: './images.component.html',
})
export class ImagesComponent {
  private readonly service = inject(ImageService);
  private readonly icons = inject(IconService);
  private readonly toast = inject(ToastService);
  private readonly confirm = inject(ConfirmService);
  private readonly destroyRef = inject(DestroyRef);
  private softRefresh: SoftRefreshHandle | null = null;
  readonly images = computed(() => this.service.cache.data() || []);
  readonly loading = this.service.cache.loading;
  readonly error = this.service.cache.error;
  readonly iconMap = computed(() => this.icons.cache.data() || {});
  readonly cleaning = signal<boolean>(false);
  readonly filter = signal('all');
  readonly filteredImages = computed(() =>
    this.images().filter(
      x =>
        this.filter() === 'all' ||
        (this.filter() === 'used'
          ? x.inUsed
          : this.filter() === 'unused'
            ? !x.inUsed
            : this.isUntagged(x)),
    ),
  );
  readonly usedCount = computed(() => this.images().filter(x => x.inUsed).length);
  readonly untaggedCount = computed(() => this.images().filter(x => this.isUntagged(x)).length);
  readonly unusedCount = computed(() => this.images().filter(x => !x.inUsed).length);
  readonly stats = computed<readonly StatItem[]>(() => [
    { key: 'all', value: this.images().length, label: '总镜像' },
    { key: 'used', value: this.usedCount(), label: '使用中', tone: 'green' },
    { key: 'unused', value: this.unusedCount(), label: '未使用', tone: 'amber' },
    { key: 'untagged', value: this.untaggedCount(), label: '无 Tag', tone: 'red' },
  ]);
  constructor() {
    this.service.ensureLoaded();
    this.icons.ensureLoaded();
    this.softRefresh = startSoftRefresh({
      intervalMs: 30_000,
      refresh: () => this.service.refresh(),
      shouldSkip: () => this.cleaning() || this.loading(),
    });
    this.destroyRef.onDestroy(() => this.softRefresh?.stop());
  }
  refresh() {
    this.service.refresh();
  }
  selectFilter(key: string): void {
    this.filter.set(this.filter() === key || key === 'all' ? 'all' : key);
  }
  isUntagged(x: ImageRow) {
    return !x.tag || ['<none>', 'none'].includes(x.tag.toLowerCase());
  }
  async cleanup(kind: 'untagged' | 'unused') {
    const count = kind === 'untagged' ? this.untaggedCount() : this.unusedCount();
    const label = kind === 'untagged' ? '无 Tag' : '未使用';
    if (
      !count ||
      !(await this.confirm.open({
        title: `清理${label}镜像`,
        message: `确定清理 ${count} 个${label}镜像吗？`,
        confirmText: '确认清理',
        danger: true,
      }))
    )
      return;
    runAction({
      request: this.service.cleanup(kind),
      onStart: () => this.cleaning.set(true),
      onFinally: () => this.cleaning.set(false),
      onSuccess: r =>
        this.toast.success(
          `已清理 ${(r.data as { deleted?: number } | undefined)?.deleted ?? ''} 个${label}镜像`,
        ),
      onBizError: r => this.toast.error(`清理失败：${r.msg || '未知错误'}`),
      onHttpError: e => this.toast.error(`清理失败：${actionErrorMessage(e)}`),
    });
  }
  icon(x: ImageRow) {
    return this.icons.resolve(x.name, this.iconMap());
  }
  fallback(e: Event) {
    (e.target as HTMLImageElement).src = this.icons.defaultIcon();
  }
  async remove(x: ImageRow, force: boolean) {
    const label = force ? '强制删除' : '删除';
    if (
      !(await this.confirm.open({
        title: `${label}镜像`,
        message: `${label}镜像 ${x.name}:${x.tag}？`,
        confirmText: label,
        danger: true,
        critical: force,
      }))
    )
      return;
    runAction({
      request: this.service.remove(x.id, force),
      onSuccess: () => this.toast.success(actionLabel(x.name, label, true)),
      onBizError: r => this.toast.error(actionLabel(x.name, label, false, r.msg || '未知错误')),
      onHttpError: e => this.toast.error(actionLabel(x.name, label, false, actionErrorMessage(e))),
    });
  }
}
