/**
 * The next keyboard focus position in a list, counting from `from` by `by` and
 * skipping items `canLand` rejects. It wraps, and returns -1 when there is
 * no item to focus.
 *
 * Counting from -1 by one returns the first item, and counting from 0 by minus
 * one returns the last.
 */
export interface RovingMotion {
  readonly from: number
  readonly by: number
}

export const stepTo = <T>(
  items: readonly T[],
  motion: RovingMotion,
  canLand: (item: T) => boolean,
): number => {
  const { from, by } = motion
  const total = items.length
  for (let step = 1; step <= total; step += 1) {
    const pos = (((from + by * step) % total) + total) % total
    const item = items[pos]
    if (item !== undefined && canLand(item)) return pos
  }
  return -1
}
