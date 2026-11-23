import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { vaultsClient } from './api/wire'
import { createSyncState } from './model/sync'
import { createSubscriptionView } from './ui/subscriptionView'
import { createSyncView } from './ui/syncStatusView'
import VaultsView from './ui/VaultsView.vue'
import './styles.css'

export const SYNC_VIEW = 'sync'
export const VAULTS_VIEW = 'vaults'
export const SUBSCRIPTION_VIEW = 'subscription'

export const activate = (plugin: PluginContext): void => {
  const state = createSyncState(vaultsClient)
  plugin.register.slotView({ id: SYNC_VIEW, component: createSyncView(state) })
  plugin.register.settingsView({ id: VAULTS_VIEW, component: VaultsView })
  plugin.register.settingsView({
    id: SUBSCRIPTION_VIEW,
    component: createSubscriptionView(plugin.identity.openExternal),
  })
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
