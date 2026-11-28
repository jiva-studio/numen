import type { PluginContext, PluginSettings } from '@numen/plugin-sdk'
import type { AccountClient } from '../api/account'
import type { SubscriptionClient } from '../api/subscription'
import type { SaveFile } from '../api/saveFile'
import type { UseSubscriptionOptions } from '../model/subscription'

export interface SubscriptionCardProps {
  client?: SubscriptionClient | undefined
  accountClient?: AccountClient | undefined
  saveFile?: SaveFile | undefined
  openExternal: PluginContext['identity']['openExternal']
  now?: (() => Date) | undefined
  settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
  openSettings?: (() => void) | undefined
  refreshMs?: UseSubscriptionOptions['refreshMs']
}
