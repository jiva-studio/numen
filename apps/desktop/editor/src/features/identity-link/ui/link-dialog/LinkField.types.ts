export interface LinkFieldProps {
  label: string
  modelValue: string
  isReadonly?: boolean
  isMono?: boolean
  isLabelHidden?: boolean
  inputmode?: 'numeric' | 'text'
  maxlength?: number
}
