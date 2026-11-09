import type { CommandItem, SlotItem } from '@numen/plugin-sdk'

export interface SlotMenuButtonProps {
  entries: readonly SlotItem[]
  label: string
  icon?: unknown
  tooltip?: string
  isItemList?: boolean
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: unknown) => void) | undefined
}
