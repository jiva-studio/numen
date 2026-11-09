/**
 * The layout of the slots as the window keeps it: read from the settings at `app.layout.slots`
 * for the plugin host, and written there in one write per move.
 */
import { computed, shallowRef } from 'vue'
import { readSlotLayout, type SlotLayout, type SlotLayoutChange } from '@numen/plugin-host'
import { write, type SettingEdit } from '@/entities/settings'

const LAYOUT_SLOTS_PATH = ['app', 'layout', 'slots'] as const

/** The settings the layout is read from and written to. */
export interface SlotLayoutSettings {
  getSetting(path: readonly string[]): unknown
  writeSettings(edits: readonly SettingEdit[]): Promise<void>
}

/** One edit per slot a move changes, each the whole new list of the slot. */
export const getLayoutEdits = (change: SlotLayoutChange): SettingEdit[] =>
  Object.entries(change).map(([slot, keys]) => ({
    at: [...LAYOUT_SLOTS_PATH, slot],
    value: write(keys),
  }))

/**
 * The layout of the window. The plugin host reads it before the settings are made, so it is empty
 * until `setSettings` hands them over, and follows them from then on.
 */
export function createWindowSlotLayout() {
  const settings = shallowRef<SlotLayoutSettings | undefined>()
  const layout = computed(() => readSlotLayout(settings.value?.getSetting(LAYOUT_SLOTS_PATH)))
  return {
    getLayout: (): SlotLayout => layout.value,
    setSettings: (source: SlotLayoutSettings): void => {
      settings.value = source
    },
    writeLayout: async (change: SlotLayoutChange): Promise<void> => {
      await settings.value?.writeSettings(getLayoutEdits(change))
    },
  }
}
