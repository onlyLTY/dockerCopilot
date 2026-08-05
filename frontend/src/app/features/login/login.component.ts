import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';
import { consumeSessionExpiredFlag } from '../../core/auth.interceptor';

@Component({
  selector: 'dc-login',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  secretKey = '';
  readonly error = signal('');
  readonly submitting = signal(false);
  readonly sessionHint = signal('');

  ngOnInit(): void {
    if (consumeSessionExpiredFlag()) {
      this.sessionHint.set('登录已过期，请重新登录后继续操作');
    }
  }

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
          this.sessionHint.set('');
          this.router.navigateByUrl(this.safeReturnUrl());
          return;
        }
        this.error.set(response.msg || '登录失败');
      },
      error: err => {
        this.submitting.set(false);
        if (err?.status === 429) {
          this.error.set(err.error?.msg || '登录尝试过于频繁，请稍后再试');
          return;
        }
        if (err?.status === 0) {
          this.error.set('无法连接服务器，请检查网络或后端是否启动');
          return;
        }
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
    if (raw === '/login' || raw.startsWith('/login?') || raw.startsWith('/login/')) {
      return '/containers';
    }
    return raw;
  }
}
