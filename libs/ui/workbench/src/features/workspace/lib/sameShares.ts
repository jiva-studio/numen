/** How close two shares are before they are the same share. */
const EPSILON = 1e-6

/** Whether two lists of shares stand at the same places. */
export const isSameShares = (a: readonly number[], b: readonly number[]): boolean =>
  a.length === b.length && a.every((share, index) => Math.abs(share - (b[index] ?? 0)) < EPSILON)
