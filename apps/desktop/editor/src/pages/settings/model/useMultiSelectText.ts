import { computed, type ComputedRef } from 'vue'
import type { SelectChoice } from '../types'

export interface MultiSelectTextSource {
  readonly choices: readonly SelectChoice[]
  readonly selected: readonly string[]
  readonly placeholder: string
  readonly getMoreText: (count: number) => string
}

/** The line of a multiple choice: the placeholder when nothing is chosen, else the first choice's text and how many more follow. */
export function useMultiSelectText(getSource: () => MultiSelectTextSource): ComputedRef<string> {
  return computed(() => {
    const { choices, selected, placeholder, getMoreText } = getSource()
    const getChosenText = (id: string): string => choices.find((one) => one.id === id)?.text || id
    const [first, ...rest] = selected
    if (first === undefined) return placeholder
    return rest.length === 0
      ? getChosenText(first)
      : `${getChosenText(first)} ${getMoreText(rest.length)}`
  })
}
