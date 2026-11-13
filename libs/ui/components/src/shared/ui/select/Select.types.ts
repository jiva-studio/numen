import type { HTMLAttributes } from 'vue'
import type { SelectChoice } from '.'

/** problem is why the choices are wrong: it marks the select and replaces the text of the empty list. */
export interface SelectProps {
  choices: readonly SelectChoice[]
  placeholder?: string
  name?: string
  isDisabled?: boolean
  isSearchable?: boolean
  canEnterCustom?: boolean
  searchName?: string
  problem?: string
  class?: HTMLAttributes['class']
}
