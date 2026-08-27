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
export { banded, landsOn, placeMenu, stepTo, MENU_OPENINGS } from './menu/model'
export type {
  BandedItem,
  MenuItem,
  MenuOpening,
  MenuPlacement,
  MenuPlacing,
} from './menu/model'

export { default as Palette } from './palette/Palette.vue'
export { default as KeyCap } from './palette/KeyCap.vue'
export { commandKeyChord, keyChord, overlayMark } from './palette/model'
export type {
  PaletteAction,
  PaletteItem,
  PaletteKeys,
  PaletteLit,
  PaletteMark,
  PalettePart,
  PaletteBand,
  PaletteSpan,
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

/** The size a node's label is being set at, and the plex drawn to hold it. */
export { optionsForType, scaleOptions, useTypeSize, DESIGNED_TYPE } from './plex/sizing'

/** How wide a title needs its box, measured against the type the page is set in. */
export { titleWidths, useTitleWidths } from './plex/measure'
export type { Measure, Measures } from './plex/measure'

export type {
  ArrangeInput,
  Direction,
  Drop,
  Placement,
  PlexOptions,
  PlexOptionsInput,
  Seating,
  Size,
  Widths,
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
export type { ActivityDescriptor, ActivityState, Counting, Tally, Tone } from './activity/model'

export { default as Notices } from './notices/Notices.vue'
export { ROOM, dwellOf, finished, folded, measured, standing, tallyOf } from './notices/model'
export type { Movement, Notice, Stay } from './notices/model'

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

export { default as Tree } from './tree/Tree.vue'

/** For arranging rows without drawing them, or reading a gesture without this renderer. */
export {
  between,
  carried,
  carries,
  everyRow,
  flatten,
  holderOf,
  isTreeKey,
  landing,
  refuses,
  sameRows,
  selects,
  stepTo as stepToRow,
  TREE_KEYS,
} from './tree/model'
export type {
  Carried,
  Landing,
  Press,
  Pressed,
  Row,
  RowId,
  ShownRow,
  Step,
  TreeKey,
} from './tree/model'

export { default as Face } from './cards/Face.vue'
export { default as Stencil } from './cards/Stencil.vue'
export { default as Deck } from './cards/Deck.vue'
export { default as Marks } from './cards/Marks.vue'
/** The strip a tile is carried by, and a box whose name sits on its outline. */
export { default as Bar } from './cards/Bar.vue'
export { default as Notched } from './cards/Notched.vue'
/** A rule with something standing in its middle. */
export { default as Rule } from './cards/Rule.vue'

/** For laying a stencil or a deck out without drawing it. */
export {
  aimedAt,
  blanks,
  faceBlocks,
  fieldRows,
  freeName,
  grid,
  laid,
  objection,
  ordered,
  panes,
  parts,
  DECK_WORDS,
  HALVES,
  SHOWS,
  STENCIL_WORDS,
} from './cards/model'
export type {
  Aim,
  Cut,
  DeckWords,
  Draft,
  Drawn,
  FaceBlock,
  FieldRow,
  Filled,
  Grid,
  Half,
  Laid,
  Landing as CardLanding,
  Objection,
  Pane,
  Part,
  Shown,
  Shows,
  StencilWords,
  Stood,
  Tile,
} from './cards/model'

/** The braces a face is written with, read and filled. */
export {
  braced,
  fill,
  insert,
  previewed,
  renamedIn,
  sampled,
  slotsIn,
  strayIn,
} from './cards/fill'
export type { Inserted, Slot } from './cards/fill'

/** What a card is written with, drawn, and measured against what a card may draw. */
export { drawn } from './cards/render'
export { safe } from './cards/safe'

export type { Environment, PlexTransition } from './plex/transition'
export type { HungPart, HungParts, PlexPart } from './plex/inside'
export type {
  EdgeArrow,
  EdgeCurve,
  EdgeHeading,
  Extent,
  PlacedArrow,
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
