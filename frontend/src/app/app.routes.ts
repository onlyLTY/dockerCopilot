import { Routes } from '@angular/router';
import { authGuard } from './core/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('./features/login/login.component').then(m => m.LoginComponent),
  },
  { path: '', pathMatch: 'full', redirectTo: 'containers' },
  {
    path: 'containers',
    loadComponent: () => import('./features/containers/containers.component').then(m => m.ContainersComponent),
    canActivate: [authGuard],
  },
  {
    path: 'images',
    loadComponent: () => import('./features/images/images.component').then(m => m.ImagesComponent),
    canActivate: [authGuard],
  },
  {
    path: 'compose',
    loadComponent: () => import('./features/compose/compose.component').then(m => m.ComposeComponent),
    canActivate: [authGuard],
  },
  {
    path: 'backups',
    loadComponent: () => import('./features/backups/backups.component').then(m => m.BackupsComponent),
    canActivate: [authGuard],
  },
  {
    path: 'ports',
    loadComponent: () => import('./features/ports/ports.component').then(m => m.PortsComponent),
    canActivate: [authGuard],
  },
  {
    path: 'about',
    loadComponent: () => import('./features/me/me.component').then(m => m.MeComponent),
    canActivate: [authGuard],
  },
  { path: 'me', redirectTo: 'about', pathMatch: 'full' },
  { path: 'settings', redirectTo: 'about', pathMatch: 'full' },
  {
    path: 'icons',
    loadComponent: () => import('./features/icons/icons.component').then(m => m.IconsComponent),
    canActivate: [authGuard],
  },
  {
    path: 'tasks',
    loadComponent: () => import('./features/tasks/tasks.component').then(m => m.TasksComponent),
    canActivate: [authGuard],
  },
  { path: '**', redirectTo: 'containers' },
];
