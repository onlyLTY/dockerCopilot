import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';

/**
 * 页面状态占位：读取中（旋转指示器）、错误、空状态。
 * 放在网格外层，作为最外层的整行占位，保证提示始终水平居中。
 */
@Component({
  selector: 'dc-page-state',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './page-state.component.html',
})
export class PageStateComponent {
  @Input() loading = false;
  @Input() error = '';
  @Input() emptyText = '暂无数据';
}
