/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { Plex } from './features/plex'
export { seatWord } from './features/plex'

export { Menu, groupItems } from './shared/ui/menu'
export type { MenuItem, MenuOpening } from './shared/ui/menu'

export { Button } from './shared/ui/button'

/** One line of digits, typed by hand and held inside its bounds. */
export { NumberField } from './shared/ui/number-field'

export { Switch } from './shared/ui/switch'

/** An hour and a minute of the day, typed on the clock the machine draws. */
export { TimeField } from './shared/ui/time-field'

/** One value along a track, moved by a handle. */
export { Slider } from './shared/ui/slider'

/** Two to four choices side by side, one of them chosen. */
export { SegmentedControl } from './shared/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './shared/ui/select'
export type { SelectChoice } from './shared/ui/select'

/** The days of the week, each drawn at the level it stands at. */
export { WeekdayChips } from './shared/ui/weekday-chips'
export { WEEK } from './shared/ui/weekday-chips'
export type { Day } from './shared/ui/weekday-chips'

export { DueCount } from './features/cards'
/** What a person did on each day, as a grid of weeks. */
export { Heatmap } from './features/heatmap'
export { dayName as heatmapDayName } from './features/heatmap'
export type { Words as HeatmapWords } from './features/heatmap'
/** A day of the calendar, written down, read back and counted against another. */
export { dayAfter, dayOf, daysBetween, getDayName, isDay } from './shared/lib/day'
export type { Tally as HeatmapTally } from './features/heatmap'
export { WelcomePage } from './features/welcome'
/** The letter a vault on that screen is opened by, and what a keystroke opens. */
export { getVaultForKey, isTyping } from './features/welcome'
export type { VaultRow, WelcomeAction } from './features/welcome'

export { Palette } from './features/palette'
export { KeyCap } from './shared/ui/key-cap'
export { commandKeyChord, keyChord } from './features/palette'
export type { ActionWords, PaletteItem, PaletteGroup } from './features/palette'
export type { PaletteKeys } from './shared/ui/key-cap'

/** The size a node's label is being set at, and the plex drawn to hold it. */
export { optionsForType, useTypeSize } from './features/plex'

export { Editor } from './features/editor'
/** A time against every line of an editor, and the line being said now. */
export { timing } from './features/editor'

/** The controls a recording is played by. What plays is somewhere else. */
export { Player } from './features/player'
export { clock } from './shared/lib/duration'

/** A stream taken up again for as long as a window is open. */
export { createFollower } from './shared/lib/stream'

export { Spinner } from './shared/ui/spinner'
export { Skeleton } from './shared/ui/skeleton'

export type { Turn } from './features/thread'
export { useConversation } from './features/thread'
export type { Conversation } from './features/thread'
export type { AgentPort, AgentStep } from './features/thread'

/** A link to a note: `[[name]]` in the text, `note://<identifier>` inside it. */
export { isNoteAddress, wikilinksIn } from './shared/lib/address'

/** A link that leads out of the application, and the window held against it. */
export { holdWindow } from './shared/lib/outward'

/** Numbers as they are read out, which both windows read the same way. */
export { many, percent, plural } from './shared/lib/digits'

export { Prose } from './shared/ui/prose'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Tone } from './features/notices'

export { Notices } from './features/notices'
export { createNotice } from './features/notices'
export type { Notice, Stay, Task } from './features/notices'

export { default as Agent } from './screens/Agent.vue'
export { Reader } from './features/reader'

/** A book made for a screen, set in columns and turned a page at a time. */
export { Book } from './features/book'
/** Where a person is reading, and what a book runs between: bytes of its text. */
export type { Span } from './features/book'

/** What a book divides into, as a list to reach any of it by. */
export { BookContents } from './features/book'
export type { ContentsEntry, ContentsWords } from './features/book'

export { WorkspaceLayout } from './features/workspace'
export { closeTab, openTab, openTabBeside } from './features/workspace'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './features/workspace'
export { paneById, panesOf } from './features/workspace'
export type { Tab, Workspace } from './features/workspace'

export { Tree } from './features/tree'
export type { Row, RowMarker } from './features/tree'

export { StencilEditor } from './features/cards'
export { DeckEditor } from './features/cards'
export { CardProse } from './features/cards'

/** The schemes a link in a card may point at. An address naming none is the caller's. */
export { scheme } from './features/cards'
/** For putting a card or a field where a person let it go, without drawing it. */
export { orderNames, reorderFields } from './features/cards'
export type { Half, InsertionPoint } from './features/cards'
/**
 * Where a card let go at the head of a deck lands, before its first section,
 * and where one let go past the last card standing under a heading lands.
 */
export {
  getBlanks as cardBlanks,
  getRunEnd,
  endOf as cardEndOf,
  HEAD as CARD_HEAD,
} from './features/cards'
export { getDeclaredFields as cardFields } from './features/cards'
export type { DeckCard, DeckSection } from './features/cards'
export type { Stencil } from './features/cards'

export { byHolding } from './features/plex'
export { byDoubleTap } from './features/plex'
export type { PlexPart } from './features/plex'
export type { EdgeArrow, PlexEdge } from './features/plex'
export type { PlexNeighbourhood } from './features/plex'
export type { PlexNode, Position } from './features/plex'
export type { PlexRelatedSeat } from './features/plex'
export type { PlexDestination } from './features/plex'
