import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';

interface ApiResponse<T> { code: number; msg: string; data: T; }
interface LoginData { jwt: string; }

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  readonly authenticated = signal(!!localStorage.getItem('docker-copilot-token'));
  login(secretKey: string): Observable<ApiResponse<LoginData>> {
    return this.http.post<ApiResponse<LoginData>>('/api/auth', { secretKey }).pipe(
      tap(response => {
        if (response.data?.jwt) {
          localStorage.setItem('docker-copilot-token', response.data.jwt);
          this.authenticated.set(true);
        }
      }),
    );
  }
  logout(): void {
    localStorage.removeItem('docker-copilot-token');
    this.authenticated.set(false);
  }
  isAuthenticated(): boolean { return this.authenticated(); }
}
