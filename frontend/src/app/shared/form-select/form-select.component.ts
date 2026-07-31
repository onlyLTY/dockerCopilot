import { Component, forwardRef, input } from '@angular/core';
import { ControlValueAccessor, FormsModule, NG_VALUE_ACCESSOR } from '@angular/forms';
import { IconComponent } from '../icon/icon.component';

export interface FormSelectOption {
  value: string;
  label: string;
}

@Component({
  selector: 'dc-form-select',
  standalone: true,
  imports: [FormsModule, IconComponent],
  templateUrl: './form-select.component.html',
  styleUrl: './form-select.component.scss',
  providers: [{ provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => FormSelectComponent), multi: true }],
})
export class FormSelectComponent implements ControlValueAccessor {
  readonly options = input<readonly FormSelectOption[]>([]);
  readonly disabled = input(false);

  protected value = '';
  protected controlDisabled = false;

  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};

  writeValue(value: string | null): void {
    this.value = value ?? '';
  }

  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.controlDisabled = disabled;
  }

  protected change(event: Event): void {
    const value = (event.target as HTMLSelectElement).value;
    this.value = value;
    this.onChange(value);
  }

  protected blur(): void {
    this.onTouched();
  }
}
