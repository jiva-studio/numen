/**
 * The entries of a slot as a reactive list, for as long as the component that reads them is
 * mounted.
 */
import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { SlotHost, SlotItem, SlotPlace } from '@numen/plugin-sdk'

export function useSlotEntries(
  host: SlotHost,
  slot: SlotPlace,
): Readonly<ShallowRef<readonly SlotItem[]>> {
  const entries = shallowRef<readonly SlotItem[]>([])
  const watching = host.watchEntries(slot, (list) => {
    entries.value = list
  })
  onScopeDispose(() => watching.dispose())
  return entries
}
