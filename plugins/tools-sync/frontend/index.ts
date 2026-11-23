import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { vaultsClient } from './api/wire'
import { createSyncState, type SyncState } from './model/sync'
import SubscriptionCard from './ui/SubscriptionCard.vue'
import SyncBadge from './ui/SyncBadge.vue'
import VaultsView from './ui/VaultsView.vue'
import './styles.css'

export const SYNC_VIEW = 'sync'
export const VAULTS_VIEW = 'vaults'
export const SUBSCRIPTION_VIEW = 'subscription'

export function createSyncView(state: SyncState) {
  return defineComponent({
    name: 'SyncStatusView',
    setup: () => () =>
      h(SyncBadge, {
        phase: state.phase.value,
      }),
  })
}

export function createSubscriptionView(openExternal: PluginContext['identity']['openExternal']) {
  return defineComponent({
    name: 'SubscriptionView',
    inheritAttrs: false,
    setup: () => () => h(SubscriptionCard, { openExternal }),
  })
}

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
