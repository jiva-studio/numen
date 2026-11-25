import { defineComponent, h } from 'vue'
import type { SyncState } from '../model/sync'
import SyncIndicator from './SyncIndicator.vue'

export function createSyncView(state: SyncState) {
  return defineComponent({
    name: 'SyncStatusView',
    setup: () => () => h(SyncIndicator, { state }),
  })
}
