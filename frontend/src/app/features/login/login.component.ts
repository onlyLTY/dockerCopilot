import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';

@Component({
  selector: 'dc-login',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  secretKey = '';
  readonly error = signal('');
  readonly submitting = signal(false);

  login(): void {
    if (this.submitting()) return;
    const key = this.secretKey.trim();
    if (!key) {
      this.error.set('请输入 secretKey');
      return;
    }
    this.error.set('');
    this.submitting.set(true);
    this.auth.login(key).subscribe({
      next: response => {
        this.submitting.set(false);
        if (response.code === 200) {
          this.router.navigateByUrl(this.safeReturnUrl());
          return;
        }
        this.error.set(response.msg || '登录失败');
      },
      error: err => {
        this.submitting.set(false);
        this.error.set(err.error?.msg ?? '登录失败');
      },
    });
  }

  /** 仅允许站内相对路径，防止开放重定向 */
  private safeReturnUrl(): string {
    const raw = this.route.snapshot.queryParamMap.get('returnUrl') || '';
    if (!raw || !raw.startsWith('/') || raw.startsWith('//') || raw.includes('://')) {
      return '/containers';
    }
    if (raw === '/login' || raw.startsWith('/login?')) {
      return '/containers';
    }
    return raw;
  }
}
