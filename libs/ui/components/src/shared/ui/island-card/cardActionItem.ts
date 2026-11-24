import type { Component } from 'vue'

/** One choice of an action that opens a menu. */
export interface CardActionChoice {
  readonly id: string
  readonly label: string
}

/** One action of a card header, as `CardActions` draws it. */
export interface CardActionItem {
  readonly id: string
  readonly label: string
  readonly icon: Component
  readonly isPrimary?: boolean
  readonly isDisabled?: boolean
  /** The label stays beside the icon at every width of the card. */
  readonly isLabelKept?: boolean
  /** The action opens a menu of these choices, and pressing one emits the id of the choice. */
  readonly choices?: readonly CardActionChoice[]
}
