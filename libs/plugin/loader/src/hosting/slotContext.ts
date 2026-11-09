/**
 * The slots in the context of a plugin: the views it registers, in the slots its manifest places
 * them, and the entries of the slots, which the owner of a slot reads and runs.
 */
import type { Disposable, SlotViewRegistration } from '@numen/plugin-sdk'
import { getDeclaredSlots, type SlotViewManifest } from '../slot-views/slotViews'
import type { PluginContextHost } from './types'

export interface SlotContextPlugin {
  readonly pluginId: string
  readonly manifest?: SlotViewManifest | undefined
}

export function createSlotContext(
  plugin: SlotContextPlugin,
  host: PluginContextHost,
  add: (disposable: Disposable) => Disposable,
) {
  const { pluginId, manifest } = plugin
  const slots = host.getSlots?.()
  return {
    ...(slots ? { slots } : {}),
    registerSlotView: (reg: SlotViewRegistration) => {
      const registrations = getDeclaredSlots(pluginId, manifest, reg.id).map((slot) =>
        host.registerSlotView(pluginId, slot, reg),
      )
      return add({ dispose: () => registrations.forEach((one) => one.dispose()) })
    },
  }
}
