import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createSyncState, SYNC_LICENSE_ID, type SyncState } from './model/sync'
import SyncBadge from './ui/SyncBadge.vue'
import './styles.css'

export const SYNC_VIEW = SYNC_LICENSE_ID

export function createSyncView(state: SyncState) {
  return defineComponent({
    name: 'SyncStatusView',
    setup: () => () =>
      h(SyncBadge, {
        phase: state.phase.value,
      }),
  })
}

export const activate = (plugin: PluginContext): void => {
  const state = createSyncState(plugin.licenses)
  plugin.register.slotView({ id: SYNC_VIEW, component: createSyncView(state) })
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
