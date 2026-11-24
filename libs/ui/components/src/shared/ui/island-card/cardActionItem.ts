import type { Component } from 'vue'

/** One action of a card header, as `CardActions` draws it. */
export interface CardActionItem {
  readonly id: string
  readonly label: string
  readonly icon: Component
  readonly isPrimary?: boolean
  readonly isDisabled?: boolean
}
