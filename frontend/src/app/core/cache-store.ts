import { signal, Signal } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

/** 缓存有效期（毫秒）：超过后即便有缓存也重新拉取，兜底防止长时间看到旧数据 */
const CACHE_TTL = 30_000;

/**
 * 通用缓存状态：把某个接口的数据、加载态、错误态缓存在 root 单例服务里，
 * 页面切换时组件被销毁重建，但服务常驻，因此缓存得以保留。
 *
 * - ensureLoaded()：组件初始化调用。缓存有效则直接用，不发请求；否则拉取。
 * - refresh()：强制拉取并覆盖缓存。手动刷新、以及某项操作成功后调用。
 * - invalidate()：仅使缓存失效（下次 ensureLoaded 会重新拉），不立即请求。
 */
export class CacheStore<T> {
  readonly data = signal<T | undefined>(undefined);
  readonly loading = signal(false);
  readonly error = signal('');

  private loadedAt = 0;

  /** 缓存是否仍然有效（已加载过且未过期） */
  private get fresh(): boolean {
    return this.data() !== undefined && Date.now() - this.loadedAt < CACHE_TTL;
  }

  constructor(
    private readonly fetcher: () => Observable<ApiResponse<T>>,
    private readonly errorText = '读取失败',
  ) {}

  /** 有缓存则不请求；无缓存或已过期则拉取 */
  ensureLoaded(): void {
    if (this.fresh) return;
    this.fetch();
  }

  /** 强制刷新并覆盖缓存 */
  refresh(): void {
    this.fetch();
  }

  /** 使缓存失效，下次进入页面会重新拉取（不立即请求） */
  invalidate(): void {
    this.loadedAt = 0;
  }

  private fetch(): void {
    this.loading.set(true);
    this.error.set('');
    this.fetcher().subscribe({
      next: r => {
        this.loading.set(false);
        if (r.code === 200 || r.code === 0) {
          this.data.set(r.data);
          this.loadedAt = Date.now();
        } else {
          this.error.set(r.msg || this.errorText);
        }
      },
      error: e => {
        this.loading.set(false);
        this.error.set(e.error?.msg || this.errorText);
      },
    });
  }
}

/** 只读视图：暴露给组件消费，避免组件直接写缓存 */
export interface CacheView<T> {
  readonly data: Signal<T | undefined>;
  readonly loading: Signal<boolean>;
  readonly error: Signal<string>;
}
