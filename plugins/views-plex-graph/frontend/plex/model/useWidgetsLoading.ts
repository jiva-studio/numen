/** Whether the views of one side of a node have loaded, told to whoever lays the nodes out. */
import { onScopeDispose, watchPostEffect } from 'vue'
import { createSlotLoading, type SlotLoading, type SlotLoadingClock } from '@numen/plugin-sdk'

/**
 * A side is ready once every view it draws has finished, and `refresh` has read what they drew, so a
 * side whose views draw something is measured before it is ready. `report` is given `null` when the side is gone.
 */
export function useWidgetsLoading(
  clock: SlotLoadingClock,
  refresh: () => void,
  report: (isReady: boolean | null) => void,
): SlotLoading {
  const loading = createSlotLoading({
    clock,
    reportIssue: (_pluginId, message) => console.warn(message),
  })

  watchPostEffect(() => {
    const isReady = loading.isReady.value
    if (isReady) refresh()
    report(isReady)
  })
  onScopeDispose(() => report(null))

  return loading
}
