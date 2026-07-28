import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../core/auth.service';

@Component({
  selector: 'dc-login',
  standalone: true,
  imports: [FormsModule, CommonModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent {
  private readonly auth = inject(AuthService); private readonly router = inject(Router); secretKey = ''; error = '';
  login(): void { this.error = ''; this.auth.login(this.secretKey).subscribe({ next: response => response.code === 200 ? this.router.navigateByUrl('/containers') : this.error = response.msg, error: err => this.error = err.error?.msg ?? '登录失败' }); }
}
