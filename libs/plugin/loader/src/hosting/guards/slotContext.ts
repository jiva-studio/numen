/**
 * The views a plugin registers, in the slots its manifest places them.
 */
import type { Disposable, SlotViewRegistration } from '@numen/plugin-sdk'
import { getDeclaredSlots, type SlotViewManifest } from '../../slot-views'
import type { PluginContextHost } from '../types'

export interface SlotContextPlugin {
  readonly pluginId: string
  readonly manifest?: SlotViewManifest | undefined
}

export function createSlotViewRegistration(
  plugin: SlotContextPlugin,
  host: PluginContextHost,
  add: (disposable: Disposable) => Disposable,
): (registration: SlotViewRegistration) => Disposable {
  const { pluginId, manifest } = plugin
  return (reg) => {
    const registrations = getDeclaredSlots(pluginId, manifest, reg.id).map((slot) =>
      host.views.registerSlotView(pluginId, slot, reg),
    )
    return add({ dispose: () => registrations.forEach((one) => one.dispose()) })
  }
}
