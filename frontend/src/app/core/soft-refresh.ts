/**
 * 页面可见时的轻量定时刷新：隐藏标签页不请求，可见时恢复可立即刷一次。
 * 用于容器/镜像等列表页，避免各组件复制 setInterval + visibility 逻辑。
 */
export interface SoftRefreshHandle {
  stop(): void;
}

export interface SoftRefreshOptions {
  /** 刷新间隔（毫秒） */
  intervalMs: number;
  /** 实际刷新动作（应内部自行去重/静默失败） */
  refresh: () => void;
  /** 返回 true 时跳过本轮（如 busy / cleaning / loading） */
  shouldSkip?: () => boolean;
  /** 切回可见时是否立即刷新，默认 true */
  refreshOnVisible?: boolean;
}

export function startSoftRefresh(options: SoftRefreshOptions): SoftRefreshHandle {
  const { intervalMs, refresh, shouldSkip, refreshOnVisible = true } = options;
  let timer: ReturnType<typeof setInterval> | null = null;

  const tick = () => {
    if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return;
    if (shouldSkip?.()) return;
    refresh();
  };

  const onVisibility = () => {
    if (!refreshOnVisible) return;
    if (typeof document === 'undefined' || document.visibilityState !== 'visible') return;
    if (shouldSkip?.()) return;
    refresh();
  };

  timer = setInterval(tick, intervalMs);
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibility);
  }

  return {
    stop() {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', onVisibility);
      }
    },
  };
}
