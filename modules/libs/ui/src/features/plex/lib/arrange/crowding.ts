import { RELATED_SEATS, type PlexRelatedSeat } from '../seat'
import { limitsFor } from './limits'
import { lerp } from './math'
import type { PlexOptions } from './options'

/** How many of each seat arrived. */
export type SeatCounts = Readonly<Record<PlexRelatedSeat, number>>

/** How finely the loosest packing is found, as halvings of the range. */
const HALVINGS = 7

/**
 * The same plex, packed to `tightness`. At one every box is drawn at its
 * widest and every gap at its setting; at nothing a box is `minWidth` and a
 * gap is the fraction of itself that `squeeze` allows.
 */
export function packOptions(options: PlexOptions, tightness: number): PlexOptions {
  const { minWidth, squeeze } = options
  return {
    ...options,
    nodeSize: {
      ...options.nodeSize,
      width: lerp(minWidth, options.nodeSize.width, tightness),
    },
    focusSize: {
      ...options.focusSize,
      width: lerp(minWidth, options.focusSize.width, tightness),
    },
    gap: lerp(options.gap * squeeze, options.gap, tightness),
    lineGap: lerp(options.lineGap * squeeze, options.lineGap, tightness),
    focusGap: lerp(options.focusGap * squeeze, options.focusGap, tightness),
  }
}

/**
 * How closely this neighbourhood has to be packed for the window to hold all
 * of it: the loosest packing that seats every node, and the closest one when
 * no packing does.
 *
 * A box narrows and a gap closes before a seat is given up. What a reader can
 * act on is a node drawn small; a node that is not drawn says only that there
 * were more.
 */
export function measureCrowding(options: PlexOptions, counts: SeatCounts): PlexOptions {
  if (!options.viewport || options.squeeze >= 1) return options
  if (canSeatAll(options, counts)) return options

  let held = 0
  let loose = 1
  for (let halving = 0; halving < HALVINGS; halving++) {
    const middle = (held + loose) / 2
    if (canSeatAll(packOptions(options, middle), counts)) held = middle
    else loose = middle
  }
  return packOptions(options, held)
}

/** Whether the window holds every node of every seat. */
function canSeatAll(options: PlexOptions, counts: SeatCounts): boolean {
  const limits = limitsFor(options, counts)
  return RELATED_SEATS.every(
    (seat) => limits[seat].perLine * limits[seat].lines >= counts[seat],
  )
}
