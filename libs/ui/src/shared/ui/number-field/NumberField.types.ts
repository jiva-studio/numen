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
  /** The words standing in for a number nobody has typed. */
  placeholder?: string
  class?: HTMLAttributes['class']
}
