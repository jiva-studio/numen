import type { PickerItem } from '@numen/plugin-sdk'
import { getTagKey } from '../../lib/tagKey'

/** The state of a tag over notes: set when every note has it, mixed when some do, unset when none does. */
export function getPickerState(
  tag: string,
  notes: readonly (readonly string[])[],
): PickerItem['state'] {
  const key = getTagKey(tag)
  const having = notes.filter((one) => one.some((found) => getTagKey(found) === key)).length
  if (having === 0) return 'unset'
  return having === notes.length ? 'set' : 'mixed'
}
