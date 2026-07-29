import { Component, input } from '@angular/core';

/**
 * 页面状态占位：读取中（旋转指示器）、错误、空状态。
 * 放在网格外层，作为最外层的整行占位，保证提示始终水平居中。
 */
@Component({
  selector: 'dc-page-state',
  standalone: true,
  templateUrl: './page-state.component.html',
})
export class PageStateComponent {
  readonly loading = input(false);
  readonly error = input('');
  readonly emptyText = input('暂无数据');
}
