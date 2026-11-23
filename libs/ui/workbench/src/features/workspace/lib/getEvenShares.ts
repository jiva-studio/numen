/** Equal shares for a branch of this many children. */
export const getEvenShares = (count: number): readonly number[] =>
  Array.from({ length: count }, () => 1 / count)
