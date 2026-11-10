import type { HTMLAttributes } from 'vue'
import type { SelectChoice } from '../../types'

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
