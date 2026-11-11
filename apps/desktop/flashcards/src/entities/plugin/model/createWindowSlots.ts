import { shallowRef } from 'vue'
import {
  createSlotRegistry,
  readManifestSlotEntries,
  type Manifest,
  type SlotViewRegistry,
} from '@numen/plugin-loader'
import { createNamedIcon } from '@numen/ui'

/** The slots the manifests of the active plugins declare, with the views the plugins draw in them. */
export function createWindowSlots(slotViews: SlotViewRegistry) {
  const manifests = shallowRef<ReadonlyMap<string, Manifest>>(new Map())
  const slots = createSlotRegistry({
    getEntries: () =>
      [...manifests.value].flatMap(([pluginId, manifest]) =>
        readManifestSlotEntries(pluginId, manifest, createNamedIcon),
      ),
    findCommand: () => undefined,
    findView: slotViews.findView,
  })
  const track = (pluginId: string, manifest: Manifest | undefined): void => {
    if (manifest) manifests.value = new Map(manifests.value).set(pluginId, manifest)
  }
  const forget = (pluginId: string): void => {
    const next = new Map(manifests.value)
    next.delete(pluginId)
    manifests.value = next
  }
  return { slots, track, forget }
}
