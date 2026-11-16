import type { PickerItem } from '@numen/plugin-sdk'

/**
 * What two spellings of one tag share, as the backend compares them: without the spaces around it and
 * a leading "#", in Unicode NFC, each segment between "/" trimmed and the empty ones dropped, and the
 * case folded.
 */
export function getTagKey(tag: string): string {
  return tag
    .trim()
    .replace(/^#/, '')
    .normalize('NFC')
    .split('/')
    .map((segment) => segment.trim())
    .filter((segment) => segment !== '')
    .join('/')
    .toLowerCase()
}

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
