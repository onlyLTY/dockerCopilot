import { HttpInterceptorFn } from '@angular/common/http';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { inject } from '@angular/core';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const token = localStorage.getItem('docker-copilot-token');
  if (!token || req.url.endsWith('/auth')) return next(req);
  return next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })).pipe(
    catchError(error => {
      if (error.status === 401) {
        auth.logout();
        if (!router.url.startsWith('/login')) router.navigateByUrl('/login');
      }
      return throwError(() => error);
    }),
  );
};
