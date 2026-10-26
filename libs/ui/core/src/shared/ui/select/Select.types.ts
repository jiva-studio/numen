import type { HTMLAttributes } from 'vue'
import type { SelectChoice } from '.'

export interface SelectProps {
  /** The choices, in the order they are offered. */
  choices: readonly SelectChoice[]
  /** What relies on the line while nothing is in force. */
  placeholder?: string
  /** What the list of choices is announced as. */
  name?: string
  isDisabled?: boolean
  /** Whether the choices carry a line to search them by. */
  isSearchable?: boolean
  /** What the search line is announced as. */
  searchName?: string
  class?: HTMLAttributes['class']
}
