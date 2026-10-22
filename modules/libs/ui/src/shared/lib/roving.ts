/**
 * Where the keyboard lands next in a list, counting from `from` by `by` and
 * passing over what `canLand` refuses. It wraps, and answers -1 when there is
 * nothing to land on.
 *
 * Counting from -1 by one is how the first is asked for, and from 0 by minus
 * one is how the last is.
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
