/**
 * Where the keyboard lands next in a list, counting from `from` by `by` and
 * passing over what `canLand` refuses. It wraps, and answers -1 when there is
 * nothing to land on.
 *
 * Counting from -1 by one is how the first is asked for, and from 0 by minus
 * one is how the last is.
 */
export const stepTo = <T>(
  items: readonly T[],
  from: number,
  by: number,
  canLand: (item: T) => boolean,
): number => {
  const total = items.length
  for (let step = 1; step <= total; step += 1) {
    const at = (((from + by * step) % total) + total) % total
    const item = items[at]
    if (item !== undefined && canLand(item)) return at
  }
  return -1
}
