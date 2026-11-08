import type { HTMLAttributes } from 'vue'

export interface NumberFieldProps {
  id?: string
  min?: number
  max?: number
  step?: number
  isDisabled?: boolean
  placeholder?: string
  class?: HTMLAttributes['class']
}
