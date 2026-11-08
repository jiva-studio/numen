import type { HTMLAttributes } from 'vue'
import type { SelectChoice } from '.'

export interface SelectProps {
  choices: readonly SelectChoice[]
  placeholder?: string
  name?: string
  isDisabled?: boolean
  isSearchable?: boolean
  canEnterCustom?: boolean
  /**
   * Whether several choices may be in force. Choosing one toggles it and the choices
   * stay open. The choices are always offered with a search line, and the values are
   * the `selected` model.
   */
  isMultiple?: boolean
  /** What the search line is announced as. */
  searchName?: string
  class?: HTMLAttributes['class']
}
