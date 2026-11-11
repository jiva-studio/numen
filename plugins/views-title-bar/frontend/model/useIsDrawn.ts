import type { SlotItem } from '@numen/plugin-sdk'

/** Whether an entry of a slot is drawn: the bar hides the entries whose keys it holds. Both are read at each ask. */
export function useIsDrawn(
  getKey: () => (entry: SlotItem) => string,
  getHidden: () => ReadonlySet<string>,
): (entry: SlotItem) => boolean {
  return (entry) => !getHidden().has(getKey()(entry))
}
