import type { HTMLAttributes } from 'vue'
import type { SelectChoice } from '.'

export interface SelectProps {
  choices: readonly SelectChoice[]
  placeholder?: string
  name?: string
  isDisabled?: boolean
  isSearchable?: boolean
  canEnterCustom?: boolean
  searchName?: string
  /** Why the choices are not what they should be: marks the select and replaces the empty list's text. */
  problem?: string
  class?: HTMLAttributes['class']
}
