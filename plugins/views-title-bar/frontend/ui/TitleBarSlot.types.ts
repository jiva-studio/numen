import type { AppInfo, CommandItem, SlotItem, Caught } from '@numen/plugin-sdk'

export interface TitleBarSlotProps {
  name: string
  entries: readonly SlotItem[]
  getKey: (entry: SlotItem) => string
  hidden: ReadonlySet<string>
  platform: AppInfo['platform']
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: Caught) => void) | undefined
}
