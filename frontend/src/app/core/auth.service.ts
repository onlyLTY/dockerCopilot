import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';

interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}
interface LoginData {
  jwt: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  readonly authenticated = signal(!!localStorage.getItem('docker-copilot-token'));
  login(secretKey: string): Observable<ApiResponse<LoginData>> {
    // 后端 LoginReq 使用 form:"secretKey"，需 application/x-www-form-urlencoded
    const body = new HttpParams().set('secretKey', secretKey);
    const headers = new HttpHeaders({ 'Content-Type': 'application/x-www-form-urlencoded' });
    return this.http.post<ApiResponse<LoginData>>('/api/auth', body, { headers }).pipe(
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
  isAuthenticated(): boolean {
    return this.authenticated();
  }
}
