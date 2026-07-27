import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';

interface ApiResponse<T> { code: number; msg: string; data: T; }
interface LoginData { jwt: string; }

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  login(secretKey: string): Observable<ApiResponse<LoginData>> {
    return this.http.post<ApiResponse<LoginData>>('/api/auth', { secretKey }).pipe(
      tap(response => {
        if (response.data?.jwt) localStorage.setItem('docker-copilot-token', response.data.jwt);
      }),
    );
  }
  logout(): void { localStorage.removeItem('docker-copilot-token'); }
  isAuthenticated(): boolean { return !!localStorage.getItem('docker-copilot-token'); }
}
