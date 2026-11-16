import type { ListItemState } from '@numen/ui'

interface PaletteMultiItem {
  readonly id: string
  readonly title: string
  readonly state: ListItemState
  readonly color?: string
}

/**
 * A step that asks for several items from a list. The ids of `recent` come most recent first.
 * `getCreateTitle` names the row that creates an item from the text typed.
 */
export interface PaletteMultiProps {
  readonly title: string
  readonly placeholder: string
  readonly items: readonly PaletteMultiItem[]
  readonly recent: readonly string[]
  readonly getCreateTitle: (text: string) => string
  readonly getCreateColor?: (text: string) => string
}
