import { Injectable, signal } from '@angular/core';
import { Observable, of, Subject } from 'rxjs';

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
  readonly authenticated = signal(true);
  readonly loggedOut = new Subject<void>();

  login(_secretKey: string): Observable<ApiResponse<LoginData>> {
    this.authenticated.set(true);
    return of({ code: 200, msg: '', data: { jwt: 'preview-token' } });
  }

  logout(): void {
    this.authenticated.set(true);
    this.loggedOut.next();
  }

  isAuthenticated(): boolean {
    return true;
  }
}
