import type { AppInfo, CommandItem, SlotItem, VaultsState, Caught } from '@numen/plugin-sdk'
import type { ButtonsSetting } from '../lib/frame'

export interface TitleBarProps {
  appName: string
  platform?: AppInfo['platform']
  buttons?: ButtonsSetting
  isMaximized?: boolean
  isFullscreen?: boolean
  vaults?: VaultsState
  commands?: readonly SlotItem[]
  status?: readonly SlotItem[]
  canCreateVault?: boolean
  listItems?: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: Caught) => void) | undefined
}
