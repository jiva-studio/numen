/**
 * The options a plex tab hands its plex: the look settings, set in the window's type.
 */
import { optionsForType, resolveOptions, type PlexOptionsInput } from '../plex'
import type { PlexLook } from '../types'

/** The look at the type size `type`, showing up to `maxHeadings` headings under a node. */
export function getPlexOptions(
  look: PlexLook,
  type: number,
  maxHeadings: number,
): PlexOptionsInput {
  const designed = resolveOptions({
    focusScale: look.focusSize.value,
    padding: { inline: look.pillPaddingSides.value, block: look.pillPaddingEnds.value },
    room: look.noteRoom.value,
    routing: { curvature: look.lineCurvature.value },
  })
  return { ...optionsForType(type, designed), maxHeadings }
}
