import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../core/auth.service';

@Component({ selector: 'dc-login', standalone: true, imports: [FormsModule, CommonModule], template: `<div class="login-page"><div class="login-card"><div class="brand centered"><div class="brand-mark">DC</div><div><strong>Docker Copilot</strong><span>容器管理平台</span></div></div><h1>登录管理台</h1><p>输入服务端配置的 secretKey 继续</p><label>Secret Key<input type="password" [(ngModel)]="secretKey" (keyup.enter)="login()" autocomplete="current-password"></label><button class="primary full" (click)="login()">进入管理台</button><div class="message error" *ngIf="error">{{ error }}</div></div></div>` })
export class LoginComponent {
  private readonly auth = inject(AuthService); private readonly router = inject(Router); secretKey = ''; error = '';
  login(): void { this.error = ''; this.auth.login(this.secretKey).subscribe({ next: response => response.code === 200 ? this.router.navigateByUrl('/dashboard') : this.error = response.msg, error: err => this.error = err.error?.msg ?? '登录失败' }); }
}
