import { onMounted, onUnmounted, shallowRef } from 'vue'
import type { SlotLayout } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { WindowClients } from '@/shared/api/clients'
import { createSettingsFollower, readSlotLayouts } from '../api/settings'

/**
 * The layout of the slots each plugin draws, which it keeps in its setting of the type slots, read
 * when the window mounts and again whenever the settings change, until it unmounts.
 */
export function useSlotLayouts(
  settings: WindowClients['settingsService'],
  wait?: (ms: number) => Promise<void>,
) {
  const layouts = shallowRef<ReadonlyMap<string, SlotLayout>>(new Map())
  const follower = createSettingsFollower(
    settings,
    async () => {
      layouts.value = await readSlotLayouts(settings)
    },
    wait,
  )

  onMounted(() => runInBackground(follower.start(), () => undefined))
  onUnmounted(follower.close)

  return {
    getSlotLayout: (pluginId: string): SlotLayout | undefined => layouts.value.get(pluginId),
  }
}
