/**
 * What a key means along a strip of tabs.
 *
 * The arrows step one along and the ends meet. Home and End go to the two
 * ends.
 */
import type { Unvalidated } from '@numen/ui'

export const stepTo = (key: string, pos: number, count: number): number | null => {
  if (pos < 0 || pos >= count) return null

  switch (key) {
    case 'ArrowRight':
      return (pos + 1) % count
    case 'ArrowLeft':
      return (pos - 1 + count) % count
    case 'Home':
      return 0
    case 'End':
      return count - 1
    default:
      return null
  }
}

/** Whether a function ref received something that takes keyboard focus. */
export const isFocusable = (value: Unvalidated): value is { focus: () => void } =>
  typeof value === 'object' &&
  value !== null &&
  'focus' in value &&
  typeof value.focus === 'function'
