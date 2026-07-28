import { Injectable } from '@angular/core';
import { CacheStore } from './cache-store';

/** 资源缓存的键，供跨页面失效使用 */
export type ResourceKey = 'containers' | 'images' | 'ports' | 'backups' | 'compose' | 'icons';

/**
 * 缓存总线：各数据服务把自己的 CacheStore 按 key 注册进来，
 * 于是某项操作后可以按 key 让相关页面的缓存失效——不需要服务互相注入（避免循环依赖）。
 */
@Injectable({ providedIn: 'root' })
export class CacheBus {
  private readonly stores = new Map<ResourceKey, CacheStore<unknown>>();

  register(key: ResourceKey, store: CacheStore<unknown>): void {
    this.stores.set(key, store);
  }

  /** 使这些资源的缓存失效：下次进入对应页面会重新拉取 */
  invalidate(keys: ResourceKey[]): void {
    keys.forEach(k => this.stores.get(k)?.invalidate());
  }

  /** 立即刷新这些资源（覆盖缓存），用于当前就展示的页面 */
  refresh(keys: ResourceKey[]): void {
    keys.forEach(k => this.stores.get(k)?.refresh());
  }
}
