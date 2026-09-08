/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { default as Plex } from './features/plex/Plex.vue'
export { seatWord } from './features/plex/seat'

export { Menu, grouped } from './shared/ui/menu'
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

export { DueCount } from './features/cards/due-count'
/** What a person did on each day, as a grid of weeks. */
export { default as Heatmap } from './features/heatmap/Heatmap.vue'
export { dayName as heatmapDayName } from './features/heatmap/dates'
export type { Words as HeatmapWords } from './features/heatmap/words'
/** A day of the calendar, written down, read back and counted against another. */
export { dayAfter, dayNamed, dayOf, daysBetween, isDay } from './shared/lib/day'
export type { Tally as HeatmapTally } from './features/heatmap/heatmap'
export { default as WelcomePage } from './features/welcome/WelcomePage.vue'
/** The letter a vault on that screen is opened by, and what a keystroke opens. */
export { opensVault, typing } from './features/welcome/letters'
export type { VaultRow, WelcomeAction } from './features/welcome/welcome'

export { default as Palette } from './features/palette/Palette.vue'
export { KeyCap } from './shared/ui/key-cap'
export { commandKeyChord, keyChord } from './features/palette/item'
export type { ActionWords, PaletteItem, PaletteGroup } from './features/palette/item'
export type { PaletteKeys } from './shared/ui/key-cap'

/** The size a node's label is being set at, and the plex drawn to hold it. */
export { optionsForType, useTypeSize } from './features/plex/sizing'

export { default as Editor } from './features/editor/Editor.vue'
/** A time against every line of an editor, and the line being said now. */
export { timing } from './features/editor/timing'

/** The controls a recording is played by. What plays is somewhere else. */
export { default as Player } from './features/player/Player.vue'
export { clock } from './shared/lib/duration'

/** A stream taken up again for as long as a window is open. */
export { following } from './shared/lib/stream'

export { Spinner } from './shared/ui/spinner'
export { Skeleton } from './shared/ui/skeleton'

export type { Turn } from './features/thread/turn'
export { conversation } from './features/thread/conversation'
export type { Conversation } from './features/thread/conversation'
export type { AgentPort, AgentStep } from './features/thread/agent'

/** A link to a note: `[[name]]` in the text, `note://<identifier>` inside it. */
export { pointsAtNote, wikilinksIn } from './shared/lib/address'

/** A link that leads out of the application, and the window held against it. */
export { holdsTheWindow } from './shared/lib/outward'

/** Numbers as they are read out, which both windows read the same way. */
export { many, percent, plural } from './shared/lib/digits'

export { Prose } from './shared/ui/prose'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Tone } from './features/notices/activity/tally'

export { default as Notices } from './features/notices/Notices.vue'
export { noticed } from './features/notices/notice'
export type { Notice, Stay, Task } from './features/notices/notice'

export { default as Agent } from './screens/Agent.vue'
export { default as Reader } from './features/reader/Reader.vue'

/** A book made for a screen, set in columns and turned a page at a time. */
export { default as Book } from './features/book/Book.vue'
/** Where a person is reading, and what a book runs between: bytes of its text. */
export type { Span as BookSpan } from './features/book/spread'

/** What a book divides into, as a list to reach any of it by. */
export { BookContents } from './features/book/book-contents'
export type { ContentsEntry, ContentsWords } from './features/book/book-contents'

export { default as WorkspaceLayout } from './features/workspace/WorkspaceLayout.vue'
export { closeTab, openTab, openTabBeside } from './features/workspace/edit'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './features/workspace/node'
export { paneById, panesOf } from './features/workspace/tree'
export type { Tab, Workspace } from './features/workspace/node'

export { default as Tree } from './features/tree/Tree.vue'
export type { Row, RowMarker } from './features/tree/row'

export { StencilEditor } from './features/cards/stencil-editor'
export { DeckEditor } from './features/cards/deck-editor'
export { CardProse } from './features/cards/card-prose'

/** The schemes a link in a card may point at. An address naming none is the caller's. */
export { scheme } from './features/cards/safe'
/** For putting a card or a field where a person let it go, without drawing it. */
export { ordered, reordered } from './features/cards/order'
export type { Half, InsertionPoint } from './features/cards/order'
/**
 * Where a card let go at the head of a deck lands, before its first section,
 * and where one let go past the last card standing under a heading lands.
 */
export { blanks as cardBlanks, ended as cardEnded, endOf as cardEndOf, HEAD as CARD_HEAD } from './features/cards/deck'
export { declared as cardFields } from './features/cards/order'
export type { DeckCard, DeckSection } from './features/cards/deck'
export type { Stencil } from './features/cards/card'

export { byHolding } from './features/plex/reaching'
export { byDoubleTap } from './features/plex/showing'
export type { PlexPart } from './features/plex/inside'
export type { EdgeArrow, PlexEdge } from './features/plex/edge'
export type { PlexNeighbourhood } from './features/plex/neighbourhood'
export type { PlexNode, Position } from './features/plex/node'
export type { PlexRelatedSeat } from './features/plex/seat'
export type { PlexShowing } from './features/plex/showing'
