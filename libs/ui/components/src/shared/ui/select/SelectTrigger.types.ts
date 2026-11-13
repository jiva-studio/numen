import type { HTMLAttributes } from 'vue'

/** isHushed draws the text as a placeholder; problem marks the line and is its title. */
export interface SelectTriggerProps {
  text: string
  isHushed: boolean
  isDisabled: boolean
  isExpanded: boolean
  hasPopup: 'menu' | 'listbox'
  problem?: string
  class?: HTMLAttributes['class']
}
