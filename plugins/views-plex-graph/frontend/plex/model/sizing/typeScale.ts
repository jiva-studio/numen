import type { PlexOptions, Size } from '../../lib/arrange'

/**
 * The size a node's label is set at with nothing multiplying it, which is the
 * chrome's type at a root of 16px. The boxes in `DEFAULT_OPTIONS` hold a label
 * of this size.
 */
export const DESIGNED_TYPE = 13

const growSize = (size: Size, delta: number): Size => ({
  width: size.width * delta,
  height: size.height * delta,
})

/**
 * The same plex, drawn larger or smaller.
 *
 * Every length is multiplied. The counts, the fractions and the directions are
 * unchanged, as is the window, which is measured in its own pixels.
 */
export const scaleOptions = (options: PlexOptions, delta: number): PlexOptions => ({
  ...options,
  focusSize: growSize(options.focusSize, delta),
  nodeSize: growSize(options.nodeSize, delta),
  padding: { inline: options.padding.inline * delta, block: options.padding.block * delta },
  room: options.room * delta,
  minWidth: options.minWidth * delta,
  iconWidth: options.iconWidth * delta,
  gap: options.gap * delta,
  lineGap: options.lineGap * delta,
  focusGap: options.focusGap * delta,
  margin: options.margin * delta,
  routing: {
    ...options.routing,
    minReach: options.routing.minReach * delta,
    arrowClearance: options.routing.arrowClearance * delta,
    minRun: options.routing.minRun * delta,
  },
})
