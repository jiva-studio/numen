import { lerp } from '../math'
import type { PlexOptions } from '../options'

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
    room: lerp(options.room * squeeze, options.room, tightness),
  }
}

/**
 * How tightly this graph must be packed for the window to hold all of it, as `canHoldAll` judges a packing: the loosest packing that fits every node, and the tightest one when no packing does.
 * A gap closes, and a box narrows unless the caller measures titles (a measured box keeps the width of its title), before a relation is dropped, since a reader can act on a node drawn small but not on one that is not drawn.
 */
export function measureCrowding(
  options: PlexOptions,
  canHoldAll: (packed: PlexOptions) => boolean,
): PlexOptions {
  if (!options.viewport || options.squeeze >= 1) return options
  if (canHoldAll(options)) return options

  let tight = 0
  let maxSqueeze = 1
  for (let halving = 0; halving < HALVINGS; halving++) {
    const middle = (tight + maxSqueeze) / 2
    if (canHoldAll(packOptions(options, middle))) tight = middle
    else maxSqueeze = middle
  }
  return packOptions(options, tight)
}
