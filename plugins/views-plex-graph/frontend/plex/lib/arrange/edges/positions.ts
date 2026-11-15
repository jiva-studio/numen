/** The middle of a line, where a title with no end is set until something is in the way. */
export const MIDDLE = 0.5

/** How far from its end a title placed near an end is set, as a fraction of the line. */
const NEAR_END = 0.2

/**
 * Where each of `count` labels of one line is set, as a fraction of the line's
 * length from its `from` end, in the order the labels are written: the first
 * is the record of the `from` note. One label is centered. Several are spread
 * from near the start to near the end, so each reads as belonging to its own end.
 */
export const getLabelPositions = (count: number): number[] => {
  if (count <= 1) return [MIDDLE]
  return Array.from(
    { length: count },
    (_, index) => NEAR_END + ((1 - 2 * NEAR_END) * index) / (count - 1),
  )
}
