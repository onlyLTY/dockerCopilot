import { Routes } from '@angular/router';
import { ComposeComponent } from './features/compose/compose.component';
import { ContainersComponent } from './features/containers/containers.component';
import { ImagesComponent } from './features/images/images.component';
import { BackupsComponent } from './features/backups/backups.component';
import { PortsComponent } from './features/ports/ports.component';
import { IconsComponent } from './features/icons/icons.component';
import { TasksComponent } from './features/tasks/tasks.component';
import { MeComponent } from './features/me/me.component';
import { LoginComponent } from './features/login/login.component';
import { authGuard } from './core/auth.guard';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: '', pathMatch: 'full', redirectTo: 'containers' },
  { path: 'containers', component: ContainersComponent, canActivate: [authGuard] },
  { path: 'images', component: ImagesComponent, canActivate: [authGuard] },
  { path: 'compose', component: ComposeComponent, canActivate: [authGuard] },
  { path: 'backups', component: BackupsComponent, canActivate: [authGuard] },
  { path: 'ports', component: PortsComponent, canActivate: [authGuard] },
  { path: 'me', component: MeComponent, canActivate: [authGuard] },
  { path: 'about', redirectTo: 'me', pathMatch: 'full' },
  { path: 'settings', redirectTo: 'me', pathMatch: 'full' },
  { path: 'icons', component: IconsComponent, canActivate: [authGuard] },
  { path: 'tasks', component: TasksComponent, canActivate: [authGuard] },
  { path: '**', redirectTo: 'containers' },
];
