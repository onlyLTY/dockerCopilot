import { Component, inject } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../core/auth.service';

@Component({
  selector: 'dc-shell', standalone: true, imports: [RouterLink, RouterLinkActive],
  template: `
  <div class="app-shell">
    <aside class="sidebar" [class.open]="open">
      <div class="brand"><div class="brand-mark">DC</div><div><strong>Docker Copilot</strong><span>容器管理平台</span></div></div>
      <nav>
        <a routerLink="/dashboard" routerLinkActive="active"><span>▦</span>仪表盘</a>
        <a routerLink="/containers" routerLinkActive="active"><span>▣</span>容器管理</a>
        <a routerLink="/compose" routerLinkActive="active"><span>◇</span>Compose 项目</a>
        <a routerLink="/ports" routerLinkActive="active"><span>↔</span>端口使用</a>
      </nav>
      <div class="sidebar-foot"><span>Docker Copilot</span><small>管理宿主机容器与项目</small></div>
    </aside>
    <div class="content-wrap">
      <header class="topbar"><button class="icon-button" (click)="open = !open">☰</button><div class="crumb">管理台 / <b>{{ title }}</b></div><button class="logout" (click)="logout()">退出</button></header>
      <main><ng-content /></main>
    </div>
  </div>`,
})
export class ShellComponent {
  private readonly auth = inject(AuthService); private readonly router = inject(Router); open = false;
  get title(): string { return this.router.url.includes('compose') ? 'Compose 项目' : this.router.url.includes('ports') ? '端口使用' : '仪表盘'; }
  logout(): void { this.auth.logout(); this.router.navigateByUrl('/login'); }
}
