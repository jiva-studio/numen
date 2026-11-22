import type { Component } from 'vue'
import type { CommandItem, SlotItem } from '@numen/plugin-sdk'
export interface TitleBarMenuButtonProps {
  entries: readonly SlotItem[]
  label: string
  icon?: Component | undefined
  tooltip?: string
  isItemList?: boolean
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: unknown) => void) | undefined
}
