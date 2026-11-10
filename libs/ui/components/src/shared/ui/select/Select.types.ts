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
  class?: HTMLAttributes['class']
}
