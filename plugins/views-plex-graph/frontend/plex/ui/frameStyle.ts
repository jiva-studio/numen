/**
 * The custom properties the drawing of a plex reads, set once on its frame.
 */
import type { PlexOptions } from '../lib/arrange'

/** The lengths of the look that are drawn in CSS and not arranged. */
export interface FrameLook {
  readonly duration: number
  readonly lineWidth: number
  readonly outlineWidth: number
}

export const getFrameStyle = (look: FrameLook, options: PlexOptions): Record<string, string> => ({
  '--plex-move': `${look.duration}ms`,
  '--plex-line': `${look.lineWidth}px`,
  '--plex-outline': `${look.outlineWidth}px`,
  '--plex-pad-inline': `${options.padding.inline}px`,
  '--plex-dot': `${options.dotSize}px`,
})
