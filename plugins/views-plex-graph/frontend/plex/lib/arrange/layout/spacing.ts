import type { PlexOptions, Size } from '../options'

/** The gaps an arrangement is laid out with. */
export interface Spacing {
  readonly gap: number
  readonly lineGap: number
  readonly focusGap: number
}

/** The gaps as they are set: the closest an arrangement is ever packed. */
export const spacingAsSet = (options: PlexOptions): Spacing => ({
  gap: options.gap,
  lineGap: options.lineGap,
  focusGap: options.focusGap,
})

/**
 * The order the gaps open in. The gap between the focus and the first line
 * goes first: it is the one an edge crosses, and an edge carries its label in
 * the middle of that crossing.
 */
const OPENS: readonly (keyof Spacing)[] = ['focusGap', 'lineGap', 'gap']

/** How finely each opening is found, as halvings of the range. */
const HALVINGS = 8

/**
 * The widest gaps for which the arrangement still fits in the window.
 *
 * `canFit` reports whether an arrangement laid out with the given gaps stays
 * inside the given window. It is supplied by the caller and not computed here,
 * so a placement of any shape is measured by the arrangement it produces.
 */
export function findSpacing(
  options: PlexOptions,
  canFit: (spacing: Spacing, viewport: Size) => boolean,
): Spacing {
  const activeSet = spacingAsSet(options)
  const { viewport } = options
  if (!viewport || options.spread <= 1 || !canFit(activeSet, viewport)) return activeSet

  let spacing = activeSet
  for (const opening of OPENS) {
    const factor = findWidestFactor(options.spread, (candidate) =>
      canFit({ ...spacing, [opening]: activeSet[opening] * candidate }, viewport),
    )
    spacing = { ...spacing, [opening]: activeSet[opening] * factor }
  }
  return spacing
}

/** The largest factor from one to `most` that holds. One is known to hold. */
function findWidestFactor(most: number, holds: (factor: number) => boolean): number {
  if (holds(most)) return most

  let low = 1
  let high = most
  for (let halving = 0; halving < HALVINGS; halving++) {
    const middle = (low + high) / 2
    if (holds(middle)) low = middle
    else high = middle
  }
  return low
}
