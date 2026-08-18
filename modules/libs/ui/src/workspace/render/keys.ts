/**
 * What a key means along a strip of tabs.
 *
 * The arrows step one along and the ends meet. Home and End go to the two
 * ends.
 */
export const stepTo = (key: string, at: number, count: number): number | null => {
  if (count < 1 || at < 0 || at >= count) return null

  switch (key) {
    case 'ArrowRight':
      return (at + 1) % count
    case 'ArrowLeft':
      return (at - 1 + count) % count
    case 'Home':
      return 0
    case 'End':
      return count - 1
    default:
      return null
  }
}
