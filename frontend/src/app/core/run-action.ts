import { Observable } from 'rxjs';

/** 与后端统一的业务响应形状（code/msg） */
export interface ActionApiResult {
  code?: number;
  msg?: string;
  data?: unknown;
}

export interface RunActionOptions<T extends ActionApiResult = ActionApiResult> {
  /** 实际请求 */
  request: Observable<T>;
  /**
   * 成功判定，默认 code === 200 或 code === 0。
   * 返回 false 时走 onBizError。
   */
  isSuccess?: (r: T) => boolean;
  /** 开始时（可选，用于 busy 标记） */
  onStart?: () => void;
  /** 结束时（成功/失败都会调用，在具体回调之后） */
  onFinally?: () => void;
  /** HTTP/网络错误 */
  onHttpError?: (err: unknown) => void;
  /** 业务失败（有响应但 code 非成功） */
  onBizError?: (r: T) => void;
  /** 业务成功 */
  onSuccess?: (r: T) => void;
}

/** 从 HttpErrorResponse 等提取可读文案 */
export function actionErrorMessage(err: unknown, fallback = '请求错误'): string {
  const e = err as
    | {
        status?: number;
        message?: string;
        error?: { msg?: string; message?: string } | string | null;
      }
    | null
    | undefined;
  if (!e) return fallback;
  if (typeof e.error === 'string' && e.error.trim()) return e.error.trim();
  if (typeof e.error === 'object' && e.error) {
    const m = e.error.msg || e.error.message;
    if (m) return String(m);
  }
  if (e.status === 0) return '无法连接服务器';
  if (e.status === 401) return '未登录或登录已过期';
  if (e.message && !/^Http failure response/i.test(e.message)) return e.message;
  return fallback;
}

/**
 * 统一「点按钮 → 请求 → toast/busy」的订阅样板，减少各页复制 next/error。
 * 不替代 forkJoin/批量流，只覆盖单次操作。
 */
export function runAction<T extends ActionApiResult>(
  options: RunActionOptions<T>,
): Promise<boolean> {
  const {
    request,
    isSuccess = r => r?.code === 200 || r?.code === 0,
    onStart,
    onFinally,
    onHttpError,
    onBizError,
    onSuccess,
  } = options;
  onStart?.();
  return new Promise<boolean>(resolve => {
    let settled = false;
    const finish = (success: boolean) => {
      if (settled) return;
      settled = true;
      try {
        onFinally?.();
      } finally {
        resolve(success);
      }
    };
    request.subscribe({
      next: r => {
        let success = false;
        try {
          success = isSuccess(r);
          if (success) onSuccess?.(r);
          else onBizError?.(r);
        } finally {
          finish(success);
        }
      },
      error: err => {
        try {
          onHttpError?.(err);
        } finally {
          finish(false);
        }
      },
      complete: () => finish(false),
    });
  });
}

/** 拼「名称 + 动作 + 成功/失败」类 toast 文案时用 */
export function actionLabel(name: string, action: string, ok: boolean, detail?: string): string {
  if (ok) return detail ? `${name} ${action}${detail}` : `${name} ${action}成功`;
  return detail ? `${name} ${action}失败：${detail}` : `${name} ${action}失败`;
}
