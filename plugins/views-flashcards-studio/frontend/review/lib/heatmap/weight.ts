export type Weight = 0 | 1 | 2 | 3 | 4

/**
 * The shade of a day. The steps are small: five answered cards is a day of
 * activity, and the grid shows it as clearly as a day of fifty.
 */
export function getWeight(did: number): Weight {
  if (did <= 0) return 0
  if (did < 5) return 1
  if (did < 20) return 2
  if (did < 50) return 3
  return 4
}
