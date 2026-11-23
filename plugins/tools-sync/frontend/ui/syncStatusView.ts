import { defineComponent, h } from 'vue'
import type { SyncState } from '../model/sync'
import SyncBadge from './SyncBadge.vue'

export function createSyncView(state: SyncState) {
  return defineComponent({
    name: 'SyncStatusView',
    setup: () => () =>
      h(SyncBadge, {
        phase: state.phase.value,
      }),
  })
}
