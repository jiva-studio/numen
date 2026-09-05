/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './tokens/theme.css'
import './window.css'

export { default as Plex } from './plex/Plex.vue'
export { browserClock } from './plex/transition'
export { isStop } from './plex/node'
export { countOf, seatWord, SEATS } from './plex/seat'
export { showingOf, SHOWINGS } from './plex/showing'

export { default as Menu } from './menu/Menu.vue'
export { banded, landsOn, placeMenu, stepTo, MENU_OPENINGS } from './menu/item'
export type { BandedItem, MenuItem, MenuOpening, MenuPlacement } from './menu/item'

export { Button, buttonVariants } from './components/ui/button'
export type { ButtonVariants } from './components/ui/button'

/** One line of digits, typed by hand and held inside its bounds. */
export { NumberField } from './components/ui/number-field'
export { clamped, numberOf, stepped, DEFAULT_BOUNDS } from './components/ui/number-field'
export type { Bounds } from './components/ui/number-field'

export { Switch } from './components/ui/switch'

/** An hour and a minute of the day, typed on the clock the machine draws. */
export { TimeField, onTheClock } from './components/ui/time-field'

/** One value along a track, moved by a handle. */
export { Slider } from './components/ui/slider'

/** Two to four choices side by side, one of them chosen. */
export { Segmented } from './components/ui/segmented'
export type { SegmentedChoice } from './components/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './components/ui/select'
export type { SelectChoice } from './components/ui/select'

/** The days of the week, each drawn at the level it stands at. */
export { Days } from './components/ui/days'
export { weekFrom, WEEK } from './components/ui/days'
export type { Day, Named } from './components/ui/days'

export { default as DueCount } from './cards/DueCount.vue'
/** What a person did on each day, as a grid of weeks. */
export { default as Heatmap } from './heatmap/Heatmap.vue'
export { dayName as heatmapDayName } from './heatmap/dates'
export type { Words as HeatmapWords } from './heatmap/words'
/** What a thing is, said beside it while a person points at it. */
export { default as Tooltip } from './tooltip/Tooltip.vue'
/** Where a thing standing over the page goes, which the menu and tooltip share. */
export { beside } from './placing/place'
export type { AxisPlacement, Box } from './placing/place'
export { days as heatmapDays, fits as heatmapFits, weighs as heatmapWeighs } from './heatmap/heatmap'
export type { Day as HeatmapDay, Room as HeatmapRoom, Tally as HeatmapTally } from './heatmap/heatmap'
export { default as Welcome } from './welcome/Welcome.vue'
export { default as Glyph } from './welcome/Glyph.vue'
/** The letter a vault on that screen is opened by, and what a keystroke opens. */
export { opensVault, typing, vaultLetter, VAULT_LETTERS } from './welcome/picking'
export type { Offer, VaultRow, Way } from './welcome/welcome'

export { default as Palette } from './palette/Palette.vue'
export { default as KeyCap } from './palette/KeyCap.vue'
export { commandKeyChord, keyChord, overlayIcon, ACTION_WORDS } from './palette/item'
export type {
  ActionWords,
  PaletteAction,
  PaletteItem,
  PaletteKeys,
  PaletteLit,
  PaletteIcon,
  PalettePart,
  PaletteBand,
  PaletteSpan,
} from './palette/item'

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
export type { Measure, PlexMetrics } from './plex/measure'

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

export { default as Editor } from './editor/Editor.vue'
export type { EditorChange } from './editor/change'
/** A time against every line of an editor, and the line being said now. */
export { timing } from './editor/timing'

/** The controls a recording is played by. What plays is somewhere else. */
export { default as Player } from './player/Player.vue'
export { clock } from './player/clock'
export type { Timed, Timing } from './editor/timing'

export { default as MessageComposer } from './composer/MessageComposer.vue'
export { COMPOSER_STATES, composerState, keyIntent } from './composer/state'
export type { ComposerState, KeyIntent } from './composer/state'

/** A stream taken up again for as long as a window is open. */
export { following } from './following/following'
export type { FollowingDeps } from './following/following'

export { default as Dots } from './dots/Dots.vue'
export { default as Spinner } from './waiting/Spinner.vue'
export { default as Skeleton } from './waiting/Skeleton.vue'

export { default as Thread } from './thread/Thread.vue'
export { VOICES, charsWord, placeTurns } from './thread/turn'
export type { PlacedTurn, Turn, TurnState, Voice } from './thread/turn'
export { conversation } from './thread/conversation'
export type { Conversation, ConversationStrings } from './thread/conversation'
export type { AgentPort, AgentStep, Place } from './thread/agent'

/** A link to a note: `[[name]]` in the text, `note://<identifier>` inside it. */
export {
  addressOf,
  pointsAtNote,
  stated,
  wikilinkAt,
  wikilinksIn,
  NAME,
  NOTE,
} from './linking/address'
export type { Address, Wikilink } from './linking/address'

/** A link that leads out of the application, and the window held against it. */
export { holdsTheWindow, pointsOutward } from './linking/outward'
export type { Opens } from './linking/outward'

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
} from './activity/tally'
export type { ActivityDescriptor, ActivityState, Tally, TallyUnit, Tone } from './activity/tally'

export { default as Notices } from './notices/Notices.vue'
export { ROOM, dwellOf, finished, folded, measured, noticed, readable, tallyOf } from './notices/notice'
export type { Movement, Notice, Stay, Task } from './notices/notice'

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
export type { NodeIdFactory, TabDrop } from './workspace/edit'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { arrangeWorkspace, DEFAULT_ARRANGE } from './workspace/arrange'
export { DEFAULT_DROP, overlayFor, sideAt, slotAt } from './workspace/drop'
export { branch, orientationAt, orientationOf, pane } from './workspace/node'
export { isBranch, isPane, paneById, panesOf, paneWithTab } from './workspace/tree'
export { normalize } from './workspace/normalize'
export type {
  Branch as WorkspaceBranchNode,
  NodeId,
  Orientation,
  Pane as WorkspacePaneNode,
  Side,
  Tab,
  TabId,
  Workspace as WorkspaceLayout,
  WorkspaceNode,
} from './workspace/node'
export type { Rect } from './workspace/rect'

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
} from './tree/row'
export type {
  DragLabel,
  Landing,
  Press,
  Row,
  RowId,
  RowMarker,
  RowSelection,
  ShownRow,
  Step,
  TreeKey,
} from './tree/row'

export { default as StencilEditor } from './cards/StencilEditor.vue'
export { default as Deck } from './cards/Deck.vue'
/** One card of a deck, which is what the deck lays out. */
export { default as Card } from './cards/Card.vue'
/** One face of a stencil, which is what the stencil lays out. */
export { default as Face } from './cards/Face.vue'
export { default as CardProse } from './cards/CardProse.vue'
/** The heading one section of a deck stands under. */
export { default as Band } from './cards/Band.vue'
/** The strip a tile is carried by. */
export { default as Bar } from './cards/Bar.vue'
/** A rule with something standing on it, in its middle or at its start. */
export { default as Divider } from './divider/Divider.vue'

/**
 * A card's HTML, measured against what a card may be drawn with. A deck may
 * come from another person, so anything drawing one goes through this.
 */
export { safe, scheme } from './cards/safe'
/** What a card is written with, drawn: markdown, with the tags among the marks. */
export { drawn } from './cards/render'
/** For putting a card or a field where a person let it go, without drawing it. */
export { ordered, reordered } from './cards/order'
export type { Half, Landing as CardLanding } from './cards/order'
/**
 * Where a card let go at the head of a deck lands, before its first section,
 * and where one let go past the last card standing under a heading lands.
 */
export { blanks as cardBlanks, ended as cardEnded, endOf as cardEndOf, HEAD as CARD_HEAD } from './cards/deck'
export { declared as cardFields } from './cards/order'
export type { DeckCard, DeckSection, FieldValue } from './cards/deck'
export type { Stencil } from './cards/stencil'

export type { Clock, PlexTransition } from './plex/transition'
export { byHandle, byHolding } from './plex/reaching'
export { APART, byDoubleClick, byDoubleTap, TAP } from './plex/showing'
export type { ShowStrategy, ShowSite } from './plex/showing'
export type { ReachStrategy, ReachSite } from './plex/reaching'
export { HOLD, STRAY, useHold } from './plex/holding'
export type { HoldState } from './plex/holding'
export type { HungPart, HungParts, PlexPart } from './plex/inside'
export type {
  EdgeArrow,
  EdgeCurve,
  EdgeHeading,
  PlacedArrow,
  PlacedEdge,
  PlexEdge,
} from './plex/edge'
export type { Extent, PlexFrame } from './plex/frame'
export type { PlexNeighbourhood } from './plex/neighbourhood'
export type { PlacedNode, PlexNode, Point } from './plex/node'
export type { PlexRelatedSeat, PlexSeat } from './plex/seat'
export type { PlexShowing, ShowingDescriptor } from './plex/showing'
