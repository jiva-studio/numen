import { RELATION_NAMES, type PlexRelation } from '../../relation'
import { getLimits } from './limits'
import { lerp } from '../math'
import type { PlexOptions } from '../options'

/** How many of each relation arrived. */
export type RelationCounts = Readonly<Record<PlexRelation, number>>

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
 * How tightly this neighbourhood must be packed for the window to hold all of it: the
 * loosest packing that fits every node, and the tightest one when no packing does.
 *
 * A box narrows and a gap closes before a relation is dropped. What a reader can act
 * on is a node drawn small; a node that is not drawn only indicates that there were
 * more.
 */
export function measureCrowding(options: PlexOptions, counts: RelationCounts): PlexOptions {
  if (!options.viewport || options.squeeze >= 1) return options
  if (canRelationAll(options, counts)) return options

  let tight = 0
  let maxSqueeze = 1
  for (let halving = 0; halving < HALVINGS; halving++) {
    const middle = (tight + maxSqueeze) / 2
    if (canRelationAll(packOptions(options, middle), counts)) tight = middle
    else maxSqueeze = middle
  }
  return packOptions(options, tight)
}

/** Whether the window holds every node of every relation. */
function canRelationAll(options: PlexOptions, counts: RelationCounts): boolean {
  const limits = getLimits(options, counts)
  return RELATION_NAMES.every(
    (relation) => limits[relation].perLine * limits[relation].lines >= counts[relation],
  )
}
