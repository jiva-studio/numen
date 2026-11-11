import { computed, type ComputedRef } from 'vue'
import type { UnavailableState } from './unavailable'

/** Whether the plugin of an unavailable tab's kind is found and the window reports it as starting. */
export function useIsStartingPlugin(getState: () => UnavailableState): ComputedRef<boolean> {
  return computed(() => {
    const state = getState()
    const lookup = state.lookup.value
    return lookup.status === 'found' && state.services.isPluginStarting?.(lookup.plugin.id) === true
  })
}
