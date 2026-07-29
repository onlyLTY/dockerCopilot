import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ShellComponent } from './layout/shell.component';
import { ToastComponent } from './shared/toast.component';
import { TaskProgressComponent } from './shared/task-progress.component';
import { ConfirmDialogComponent } from './shared/confirm-dialog.component';

@Component({ selector: 'dc-root', standalone: true, imports: [RouterOutlet, ShellComponent, ToastComponent, TaskProgressComponent, ConfirmDialogComponent], template: `<dc-shell><router-outlet /></dc-shell><dc-toast /><dc-task-progress /><dc-confirm-dialog />` })
export class AppComponent {}
