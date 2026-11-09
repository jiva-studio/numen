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

export interface MultiSelectProps {
  choices: readonly SelectChoice[]
  selected: readonly string[]
  placeholder?: string
  name?: string
  isDisabled?: boolean
  searchName?: string
  getMoreText?: (count: number) => string
  class?: HTMLAttributes['class']
}
