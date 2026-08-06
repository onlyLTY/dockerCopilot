import { HttpInterceptorFn } from '@angular/common/http';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { inject } from '@angular/core';
import { AuthService } from './auth.service';
import { ToastService } from './toast.service';

const SESSION_EXPIRED_FLAG = 'dc-session-expired';

/** 供登录页读取并清除：是否因会话过期被踢回 */
export function consumeSessionExpiredFlag(): boolean {
  try {
    if (sessionStorage.getItem(SESSION_EXPIRED_FLAG) === '1') {
      sessionStorage.removeItem(SESSION_EXPIRED_FLAG);
      return true;
    }
  } catch {
    /* ignore */
  }
  return false;
}

function markSessionExpired(): void {
  try {
    sessionStorage.setItem(SESSION_EXPIRED_FLAG, '1');
  } catch {
    /* ignore */
  }
}

/** 当前 URL 转为登录 returnUrl（仅站内相对路径） */
function currentReturnUrl(router: Router): string {
  const tree = router.parseUrl(router.url);
  // 已在登录页时不要套娃
  if (tree.root.children['primary']?.segments[0]?.path === 'login') {
    return '/containers';
  }
  const url = router.url || '/containers';
  if (!url.startsWith('/') || url.startsWith('//') || url.includes('://')) {
    return '/containers';
  }
  return url;
}

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const toast = inject(ToastService);
  const token = localStorage.getItem('docker-copilot-token');
  if (!token || req.url.includes('/api/auth')) return next(req);
  return next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })).pipe(
    catchError(error => {
      // 仅处理「已带 token 仍 401」：会话过期或 secretKey 变更
      if (error.status === 401 && auth.isAuthenticated()) {
        auth.logout();
        markSessionExpired();
        if (!router.url.startsWith('/login')) {
          toast.error('登录已过期', '请重新登录后继续操作');
          const returnUrl = currentReturnUrl(router);
          void router.navigate(['/login'], { queryParams: { returnUrl } });
        }
      }
      return throwError(() => error);
    }),
  );
};
