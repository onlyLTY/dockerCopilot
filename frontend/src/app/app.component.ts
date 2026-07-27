import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ShellComponent } from './layout/shell.component';

@Component({ selector: 'dc-root', standalone: true, imports: [RouterOutlet, ShellComponent], template: `<dc-shell><router-outlet /></dc-shell>` })
export class AppComponent {}
