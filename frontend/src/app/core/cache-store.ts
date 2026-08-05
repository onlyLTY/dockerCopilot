import { computed, signal, Signal } from '@angular/core';
import { Observable, Subscription } from 'rxjs';
import { ApiResponse } from './compose.service';

/** 默认缓存有效期（毫秒）：超过后即便有缓存也重新拉取，兜底防止长时间看到旧数据 */
const DEFAULT_CACHE_TTL = 30_000;

export interface CacheStoreOptions {
  /** 缓存 TTL（毫秒）。0 表示仅在 invalidate/refresh 时失效，适合图标等变更低频数据 */
  ttlMs?: number;
}

/**
 * 通用缓存状态：把某个接口的数据、加载态、错误态缓存在 root 单例服务里，
 * 页面切换时组件被销毁重建，但服务常驻，因此缓存得以保留。
 *
 * - ensureLoaded()：组件初始化调用。缓存有效则直接用，不发请求；否则拉取。
 * - refresh()：强制拉取并覆盖缓存。手动刷新、以及某项操作成功后调用。
 * - invalidate()：仅使缓存过期标记失效（下次 ensureLoaded 会重新拉），不立即请求、不清空已有数据。
 * - 进行中的请求会去重，避免 refresh 风暴重复打 API。
 * - settled：至少成功拿到过一次响应后为 true；未 settled 前 loading 恒为 true，避免先闪「暂无」。
 */
export class CacheStore<T> {
  readonly data = signal<T | undefined>(undefined);
  readonly error = signal('');

  /** 是否已成功完成至少一次加载（含空列表） */
  private readonly settled = signal(false);
  private loadedAt = 0;
  private inflight: Subscription | null = null;
  private readonly ttlMs: number;

  /**
   * 页面空态用的加载中：
   * - 尚未成功拿到过数据 → 显示 spinner，绝不先渲染「暂无」
   * - 已 settled 后的后台/过期刷新 → 保持旧数据（或空列表）展示，不闪 spinner
   * - 有错误且未 settled → 交给 error 展示
   */
  readonly loading: Signal<boolean> = computed(() => !this.settled() && !this.error());

  /** 缓存是否仍然有效（已加载过且未过期；ttlMs=0 时 settled 即视为有效） */
  private get fresh(): boolean {
    if (!this.settled()) return false;
    if (this.ttlMs <= 0) return true;
    return Date.now() - this.loadedAt < this.ttlMs;
  }

  constructor(
    private readonly fetcher: () => Observable<ApiResponse<T>>,
    private readonly errorText = '读取失败',
    options?: CacheStoreOptions,
  ) {
    this.ttlMs = options?.ttlMs ?? DEFAULT_CACHE_TTL;
  }

  /** 有缓存则不请求；无缓存或已过期则拉取 */
  ensureLoaded(): void {
    if (this.fresh || this.inflight) return;
    this.fetch();
  }

  /** 强制刷新并覆盖缓存（若已有进行中请求则忽略重复触发） */
  refresh(): void {
    if (this.inflight) return;
    this.fetch();
  }

  /** 使缓存过期，下次进入页面会重新拉取（不立即请求、不丢旧数据） */
  invalidate(): void {
    this.loadedAt = 0;
  }

  private fetch(): void {
    // 首次（未 settled）才清错误并进入整页 loading；已有数据的静默刷新保留当前视图
    if (!this.settled()) this.error.set('');

    this.inflight = this.fetcher().subscribe({
      next: r => {
        this.inflight = null;
        // 部分接口可能返回空 body（null），需兜底避免读 code 崩溃
        if (!r) {
          if (!this.settled()) this.error.set(this.withReason('响应为空'));
          return;
        }
        if (r.code === 200 || r.code === 0) {
          this.data.set(r.data as T);
          this.loadedAt = Date.now();
          this.settled.set(true);
          this.error.set('');
        } else if (!this.settled()) {
          // 有旧数据时静默失败；未就绪时才展示错误
          this.error.set(this.withReason(r.msg || `业务错误码 ${r.code}`));
        }
      },
      error: e => {
        this.inflight = null;
        if (!this.settled()) this.error.set(this.formatHttpError(e));
      },
    });
  }

  /** 统一成「读取xxx失败：原因」 */
  private withReason(reason: string): string {
    const text = (reason || '').trim();
    return text ? `${this.errorText}：${text}` : this.errorText;
  }

  /** 从 HttpErrorResponse / 网络异常中提炼可读原因 */
  private formatHttpError(e: unknown): string {
    const err = e as {
      status?: number;
      statusText?: string;
      message?: string;
      error?: { msg?: string; message?: string } | string | null;
    } | null | undefined;
    if (!err) return this.errorText;

    const bodyMsg =
      (typeof err.error === 'string' && err.error.trim()) ||
      (typeof err.error === 'object' && err.error && (err.error.msg || err.error.message)) ||
      '';
    const status = typeof err.status === 'number' ? err.status : undefined;

    if (status === 0) return this.withReason('无法连接服务器，请检查网络或后端是否启动');
    if (status === 401) return this.withReason('未登录或登录已过期');
    if (status === 403) return this.withReason('没有权限访问');
    if (status === 404) return this.withReason('接口不存在 (404)');
    if (status === 502 || status === 503 || status === 504) {
      return this.withReason(`服务暂时不可用 (${status})`);
    }
    if (bodyMsg) return this.withReason(String(bodyMsg));
    if (status && status > 0) {
      const text = (err.statusText || '').trim();
      return this.withReason(text ? `${text} (${status})` : `HTTP ${status}`);
    }
    const raw = (err.message || '').trim();
    if (raw && !/^Http failure response/i.test(raw)) return this.withReason(raw);
    return this.errorText;
  }
}

/** 只读视图：暴露给组件消费，避免组件直接写缓存 */
export interface CacheView<T> {
  readonly data: Signal<T | undefined>;
  readonly loading: Signal<boolean>;
  readonly error: Signal<string>;
}
