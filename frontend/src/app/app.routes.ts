import { Routes } from '@angular/router';
import { DashboardComponent } from './features/dashboard/dashboard.component';
import { ComposeComponent } from './features/compose/compose.component';
import { LoginComponent } from './features/login.component';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
  { path: 'dashboard', component: DashboardComponent },
  { path: 'containers', component: DashboardComponent },
  { path: 'compose', component: ComposeComponent },
  { path: 'ports', component: ComposeComponent },
  { path: '**', redirectTo: 'dashboard' },
];
