import type { Component } from 'vue'

/** One choice of an action that opens a menu. */
export interface CardActionChoice {
  readonly id: string
  readonly label: string
  readonly detail?: string
  readonly icon?: Component
}

/** One action of a card header, as `CardActions` draws it. */
export interface CardActionItem {
  readonly id: string
  readonly label: string
  readonly icon: Component
  readonly isPrimary?: boolean
  readonly isDisabled?: boolean
  readonly isLabelKept?: boolean
  readonly choices?: readonly CardActionChoice[]
}
