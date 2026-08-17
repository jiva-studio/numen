/**
 * Shares of a branch's length: one per child, summing to one.
 *
 * Every function here returns shares in that form, so a caller may hand in
 * whatever it has and use what comes back directly.
 */

/** Equal shares for a branch of this many children. */
export const even = (count: number): readonly number[] =>
  count > 0 ? Array.from({ length: count }, () => 1 / count) : []

/**
 * The same shares, one per child and summing to one.
 *
 * A share that is missing or not a positive number takes the average of the
 * ones that are, so a child brought in without a size is given room.
 */
export function fit(sizes: readonly number[], count: number): readonly number[] {
  if (count <= 0) return []

  const given: number[] = []
  for (let i = 0; i < count; i++) {
    const size = sizes[i]
    given.push(typeof size === 'number' && Number.isFinite(size) && size > 0 ? size : 0)
  }

  const total = given.reduce((sum, size) => sum + size, 0)
  if (total <= 0) return even(count)

  const missing = given.filter((size) => size === 0).length
  if (missing === 0) return given.map((size) => size / total)

  const average = total / (count - missing)
  const filled = given.map((size) => (size > 0 ? size : average))
  const whole = filled.reduce((sum, size) => sum + size, 0)
  return filled.map((size) => size / whole)
}

/** Room made beside the share at `index`, taking half of it. */
export function insert(
  sizes: readonly number[],
  index: number,
  before: boolean,
): readonly number[] {
  const fitted = [...fit(sizes, sizes.length)]
  const half = (fitted[index] ?? 1 / Math.max(fitted.length, 1)) / 2
  fitted[index] = half
  fitted.splice(before ? index : index + 1, 0, half)
  return fit(fitted, fitted.length)
}

/** The shares left once the one at `index` is gone, its length shared out. */
export function remove(sizes: readonly number[], index: number): readonly number[] {
  const fitted = [...fit(sizes, sizes.length)]
  fitted.splice(index, 1)
  return fit(fitted, fitted.length)
}

/** The share at `index` divided among `inner`, in proportion to them. */
export function spread(
  sizes: readonly number[],
  index: number,
  inner: readonly number[],
): readonly number[] {
  const fitted = [...fit(sizes, sizes.length)]
  const share = fitted[index] ?? 0
  const parts = fit(inner, inner.length).map((part) => part * share)
  fitted.splice(index, 1, ...parts)
  return fit(fitted, fitted.length)
}
