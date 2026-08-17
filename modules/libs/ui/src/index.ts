/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './tokens/theme.css'

export { default as Plex } from './plex/Plex.vue'
export { browserEnvironment } from './plex/transition'
export { countOf, isStop, seatWord, SEATS } from './plex/model'

export { default as Menu } from './menu/Menu.vue'
export { placeMenu, stepTo } from './menu/model'
export type { MenuItem, MenuPlacement, MenuPlacing } from './menu/model'

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

export { default as Editor } from './editor/Editor.vue'

export { default as Composer } from './composer/Composer.vue'
export { COMPOSER_STATES, composerState, keyIntent } from './composer/model'
export type { ComposerState, KeyIntent } from './composer/model'

export { default as Dots } from './dots/Dots.vue'

export { default as Thread } from './thread/Thread.vue'
export { VOICES, VOICE_NAMES, charsWord, placeTurns } from './thread/model'
export type { PlacedTurn, Turn, TurnState, Voice } from './thread/model'

export { default as Prose } from './prose/Prose.vue'
export { default as Tool } from './tool/Tool.vue'

export { default as Activity } from './activity/Activity.vue'
export { activity, percentWord, rateOf, remainingWord, shareOf, tallyWord } from './activity/model'
export type { ActivityDescriptor, ActivityState, Tally } from './activity/model'

export { default as Agent } from './assembled/Agent.vue'

export { default as Workspace } from './workspace/Workspace.vue'
export {
  activateTab,
  closeTab,
  dropOnEdge,
  dropTab,
  focusPane,
  moveTabWithin,
  openTab,
  resizeBranch,
} from './workspace/edit'
export type { Naming, TabDrop } from './workspace/edit'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { arrangeWorkspace, DEFAULT_ARRANGE } from './workspace/arrange'
export { DEFAULT_DROP, overlayFor, sideAt, slotAt } from './workspace/drop'
export {
  branch,
  isBranch,
  isPane,
  normalize,
  orientationAt,
  orientationOf,
  pane,
  paneById,
  panesOf,
  paneWithTab,
} from './workspace/model'
export type {
  Branch as WorkspaceBranchNode,
  NodeId,
  Orientation,
  Pane as WorkspacePaneNode,
  Rect,
  Side,
  Tab,
  TabId,
  Workspace as WorkspaceLayout,
  WorkspaceNode,
} from './workspace/model'

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
