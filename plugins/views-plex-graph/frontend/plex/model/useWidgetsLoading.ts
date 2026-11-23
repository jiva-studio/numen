/** Whether the views of one side of a node have loaded and what they drew is measured, reported to the layout of the nodes. */
import { onScopeDispose, watchPostEffect, type Ref } from 'vue'
import { createSlotLoading, type SlotLoading, type SlotLoadingClock } from '@numen/plugin-sdk'

/**
 * A side is ready once every view it draws has finished and `isMeasured` holds after `refresh` has read what they drew.
 * `report` receives `null` when the side is removed.
 */
export function useWidgetsLoading(
  clock: SlotLoadingClock,
  drawn: { isMeasured: Readonly<Ref<boolean>>; refresh: () => void },
  report: (isReady: boolean | null) => void,
): SlotLoading {
  const slotLoading = createSlotLoading({ clock })

  watchPostEffect(() => {
    const isLoaded = slotLoading.isReady.value
    if (isLoaded) drawn.refresh()
    report(isLoaded && drawn.isMeasured.value)
  })
  onScopeDispose(() => report(null))

  return slotLoading
}
