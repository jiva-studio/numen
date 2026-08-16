/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './tokens/tokens.css'

export { default as Plex } from './plex/Plex.vue'
export { browserEnvironment } from './plex/transition'
export { countOf, seatWord, SEATS } from './plex/model'

/** For arranging without drawing, or drawing without this renderer. */
export { default as PlexView } from './plex/render/PlexView.vue'
export {
  arrangePlex,
  interpolatePlex,
  rowsAndColumns,
  DEFAULT_OPTIONS,
  DEFAULT_DIRECTION,
} from './plex/arrange'

export type {
  ArrangeInput,
  Direction,
  Drop,
  Placement,
  PlexOptions,
  PlexOptionsInput,
  Seating,
  Size,
} from './plex/arrange'
export { default as Panel } from './panel/Panel.vue'

export { default as Composer } from './composer/Composer.vue'
export { COMPOSER_STATES, composerState, keyIntent } from './composer/model'
export type { ComposerState, KeyIntent } from './composer/model'

export { default as Dots } from './dots/Dots.vue'

export { default as Thread } from './thread/Thread.vue'
export { VOICES, VOICE_NAMES, placeTurns } from './thread/model'
export type { PlacedTurn, Turn, TurnState, Voice } from './thread/model'

export type { Environment, PlexTransition } from './plex/transition'
export type {
  Extent,
  PlacedEdge,
  PlacedNode,
  PlexEdge,
  PlexFrame,
  PlexNeighbourhood,
  PlexNode,
  PlexRelatedSeat,
  PlexSeat,
  Point,
} from './plex/model'
