import type { Component } from 'vue'
import type { CommandItem, SlotItem, Caught } from '@numen/plugin-sdk'

export interface SlotMenuButtonProps {
  entries: readonly SlotItem[]
  label: string
  icon?: Component | undefined
  tooltip?: string
  isItemList?: boolean
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: Caught) => void) | undefined
}
