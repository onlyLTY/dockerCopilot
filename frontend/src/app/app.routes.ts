import { Routes } from '@angular/router';
import { ComposeComponent } from './features/compose/compose.component';
import { ContainersComponent } from './features/containers/containers.component';
import { ImagesComponent } from './features/images/images.component';
import { BackupsComponent } from './features/backups/backups.component';
import { PortsComponent } from './features/ports/ports.component';
import { AboutComponent } from './features/about/about.component';
import { IconsComponent } from './features/icons/icons.component';
import { TasksComponent } from './features/tasks/tasks.component';
import { SettingsComponent } from './features/settings/settings.component';
import { LoginComponent } from './features/login.component';
import { authGuard } from './core/auth.guard';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: '', pathMatch: 'full', redirectTo: 'containers' },
  { path: 'containers', component: ContainersComponent, canActivate: [authGuard] },
  { path: 'images', component: ImagesComponent, canActivate: [authGuard] },
  { path: 'compose', component: ComposeComponent, canActivate: [authGuard] },
  { path: 'backups', component: BackupsComponent, canActivate: [authGuard] },
  { path: 'ports', component: PortsComponent, canActivate: [authGuard] },
  { path: 'about', component: AboutComponent, canActivate: [authGuard] },
  { path: 'icons', component: IconsComponent, canActivate: [authGuard] },
  { path: 'tasks', component: TasksComponent, canActivate: [authGuard] },
  { path: 'settings', component: SettingsComponent, canActivate: [authGuard] },
  { path: '**', redirectTo: 'containers' },
];
