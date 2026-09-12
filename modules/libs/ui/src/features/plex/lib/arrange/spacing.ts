import type { PlexOptions } from './options'

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
 * The widest gaps the window still holds.
 *
 * `canFit` answers whether an arrangement laid out with the given gaps stays
 * inside the window. It is asked, not worked out here, so a placement of any
 * shape is measured by the arrangement it produces.
 */
export function spacingFor(
  options: PlexOptions,
  canFit: (spacing: Spacing) => boolean,
): Spacing {
  const set = spacingAsSet(options)
  if (!options.viewport || options.spread <= 1 || !canFit(set)) return set

  let spacing = set
  for (const opening of OPENS) {
    const factor = widest(options.spread, (candidate) =>
      canFit({ ...spacing, [opening]: set[opening] * candidate }),
    )
    spacing = { ...spacing, [opening]: set[opening] * factor }
  }
  return spacing
}

/** The largest factor from one to `most` that holds. One is known to hold. */
function widest(most: number, holds: (factor: number) => boolean): number {
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
