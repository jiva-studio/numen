/**
 * What a key means along a strip of tabs.
 *
 * The arrows step one along and the ends meet. Home and End go to the two
 * ends.
 */
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
