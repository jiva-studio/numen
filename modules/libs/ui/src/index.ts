/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './tokens/theme.css'

export { default as Plex } from './plex/Plex.vue'
export { browserEnvironment } from './plex/transition'
export { countOf, isStop, seatWord, showingOf, SEATS, SHOWINGS } from './plex/model'

export { default as Menu } from './menu/Menu.vue'
export { landsOn, placeMenu, stepTo, MENU_OPENINGS } from './menu/model'
export type { MenuItem, MenuOpening, MenuPlacement, MenuPlacing } from './menu/model'

export { default as Palette } from './palette/Palette.vue'
export { commandKeyWord, keptOn, PALETTE_KEYS } from './palette/model'
export type {
  PaletteAction,
  PaletteItem,
  PaletteKeyed,
  PalettePart,
  PaletteBand,
  PaletteSpan,
  PlacedAction,
} from './palette/model'

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
export type { EditorChange } from './editor/change'

export { default as Composer } from './composer/Composer.vue'
export { COMPOSER_STATES, composerState, keyIntent } from './composer/model'
export type { ComposerState, KeyIntent } from './composer/model'

export { default as Dots } from './dots/Dots.vue'
export { default as Waiting } from './waiting/Waiting.vue'

export { default as Thread } from './thread/Thread.vue'
export { VOICES, VOICE_NAMES, charsWord, placeTurns } from './thread/model'
export type { PlacedTurn, Turn, TurnState, Voice } from './thread/model'

export { default as Prose } from './prose/Prose.vue'
export { default as Tool } from './tool/Tool.vue'

export { default as Activity } from './activity/Activity.vue'
export {
  activity,
  percentWord,
  rateOf,
  rateWord,
  remainingWord,
  shareOf,
  sizeWord,
  tallyWord,
} from './activity/model'
export type { ActivityDescriptor, ActivityState, Counting, Tally } from './activity/model'

export { default as Notices } from './notices/Notices.vue'
export { measured, standing, tallyOf } from './notices/model'
export type { Movement, Notice } from './notices/model'

export { default as Agent } from './screens/Agent.vue'
export { default as Reader } from './reader/Reader.vue'

export { default as Workspace } from './workspace/Workspace.vue'
export {
  activateTab,
  closeTab,
  dropOnEdge,
  dropTab,
  focusPane,
  moveTabWithin,
  openTab,
  openTabBeside,
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
  PlexShowing,
  Point,
  ShowingDescriptor,
} from './plex/model'
