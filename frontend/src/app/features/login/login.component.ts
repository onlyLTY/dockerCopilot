import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';

@Component({
  selector: 'dc-login',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.scss',
})
export class LoginComponent {
  private readonly auth = inject(AuthService); private readonly router = inject(Router); secretKey = ''; readonly error = signal('');
  login(): void { this.error.set(''); this.auth.login(this.secretKey).subscribe({ next: response => response.code === 200 ? this.router.navigateByUrl('/containers') : this.error.set(response.msg), error: err => this.error.set(err.error?.msg ?? '登录失败') }); }
}
