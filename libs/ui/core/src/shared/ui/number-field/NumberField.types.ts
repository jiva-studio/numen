import type { HTMLAttributes } from 'vue'

export interface NumberFieldProps {
  /** What a `<label for>` names the field by. Left empty, the field makes up its own. */
  id?: string
  /** How far the number may go. */
  min?: number
  max?: number
  /** What an arrow key moves it by. */
  step?: number
  isDisabled?: boolean
  /** Text shown in place of a number when no value has been entered. */
  placeholder?: string
  class?: HTMLAttributes['class']
}
