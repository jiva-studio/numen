import {
  definePlugin,
  SETTINGS_OPEN_COMMAND,
  type HostServices,
  type PluginContext,
} from '@numen/plugin-sdk'
import { vaultsClient } from './api/wire'
import { createSyncState } from './model/sync'
import { createSubscriptionView } from './ui/subscriptionView'
import { createSyncView } from './ui/syncStatusView'
import VaultsView from './ui/VaultsView.vue'
import './styles.css'

export const SYNC_VIEW = 'sync'
export const VAULTS_VIEW = 'vaults'
export const SUBSCRIPTION_VIEW = 'subscription'

const createSettingsOpener = (
  plugin: PluginContext,
  host: Pick<HostServices, 'ui'> | undefined,
): (() => void) | undefined => {
  const commands = host?.ui.commands
  if (commands === undefined) return undefined
  return () => {
    commands
      .run(SETTINGS_OPEN_COMMAND, undefined, undefined, { plugin: plugin.identity.id })
      .catch((error: unknown) => host?.ui.notices.show(String(error), 'error'))
  }
}

export const activate = (plugin: PluginContext, host?: Pick<HostServices, 'ui'>): void => {
  const state = createSyncState(vaultsClient)
  plugin.register.slotView({ id: SYNC_VIEW, component: createSyncView(state) })
  plugin.register.settingsView({ id: VAULTS_VIEW, component: VaultsView })
  plugin.register.settingsView({
    id: SUBSCRIPTION_VIEW,
    component: createSubscriptionView(plugin, createSettingsOpener(plugin, host)),
  })
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
