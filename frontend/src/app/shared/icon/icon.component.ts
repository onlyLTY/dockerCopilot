import { Component, computed, input } from '@angular/core';

/**
 * dc-icon 使用 CSS mask 渲染 assets/icons 下的 SVG，
 * 图标颜色由 CSS `color`（currentColor）决定，因此会自动跟随所在按钮/链接的文字颜色。
 * 通过 `--icon-size` 变量控制尺寸。
 */
@Component({
  selector: 'dc-icon',
  standalone: true,
  template: '',
  host: {
    '[style.-webkit-mask-image]': 'maskUrl()',
    '[style.mask-image]': 'maskUrl()',
  },
  styleUrl: './icon.component.scss',
})
export class IconComponent {
  readonly name = input.required<string>();
  readonly maskUrl = computed(() => `url(assets/icons/${this.name()}.svg)`);
}
