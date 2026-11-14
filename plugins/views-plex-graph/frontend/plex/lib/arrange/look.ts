/**
 * The look settings, as the boxes and gaps of an arrangement.
 */
import type { Size } from '@numen/ui'

/** The space between a title and the edges of its pill. */
export interface Padding {
  readonly inline: number
  readonly block: number
}

interface Gaps {
  readonly gap: number
  readonly lineGap: number
  readonly focusGap: number
  readonly room: number
}

/**
 * The height of one line of a title at the designed type: 13 pixels set at a line height
 * of 1.35.
 */
const TITLE_LINE = 13 * 1.35

/** The box of a neighbor: a line of title between the top and bottom padding. */
export const getNodeSize = (width: number, padding: Padding): Size => ({
  width,
  height: TITLE_LINE + 2 * padding.block,
})

export const getFocusSize = (nodeSize: Size, focusScale: number): Size => ({
  width: nodeSize.width * focusScale,
  height: nodeSize.height * focusScale,
})

/**
 * The gaps widened by the room on both sides of a box. The arrangement applies it once,
 * before it packs a crowded window, so the room closes as the gaps do.
 */
export const applyRoom = <T extends Gaps>(options: T): T =>
  options.room === 0
    ? options
    : {
        ...options,
        gap: options.gap + 2 * options.room,
        lineGap: options.lineGap + 2 * options.room,
        focusGap: options.focusGap + 2 * options.room,
      }
