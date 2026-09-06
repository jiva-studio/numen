/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './tokens/theme.css'
import './window.css'

export { default as Plex } from './plex/Plex.vue'
export { seatWord } from './plex/seat'

export { default as Menu } from './menu/Menu.vue'
export { grouped } from './menu/item'
export type { MenuItem, MenuOpening } from './menu/item'

export { Button } from './components/ui/button'

/** One line of digits, typed by hand and held inside its bounds. */
export { NumberField } from './components/ui/number-field'

export { Switch } from './components/ui/switch'

/** An hour and a minute of the day, typed on the clock the machine draws. */
export { TimeField } from './components/ui/time-field'

/** One value along a track, moved by a handle. */
export { Slider } from './components/ui/slider'

/** Two to four choices side by side, one of them chosen. */
export { SegmentedControl } from './components/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './components/ui/select'
export type { SelectChoice } from './components/ui/select'

/** The days of the week, each drawn at the level it stands at. */
export { WeekdayChips } from './components/ui/weekday-chips'
export { WEEK } from './components/ui/weekday-chips'
export type { Day } from './components/ui/weekday-chips'

export { default as DueCount } from './cards/DueCount.vue'
/** What a person did on each day, as a grid of weeks. */
export { default as Heatmap } from './heatmap/Heatmap.vue'
export { dayName as heatmapDayName } from './heatmap/dates'
export type { Words as HeatmapWords } from './heatmap/words'
/** A day of the calendar, written down, read back and counted against another. */
export { dayAfter, dayNamed, dayOf, daysBetween, isDay } from './calendar/day'
export type { Tally as HeatmapTally } from './heatmap/heatmap'
export { default as WelcomePage } from './welcome/WelcomePage.vue'
/** The letter a vault on that screen is opened by, and what a keystroke opens. */
export { opensVault, typing } from './welcome/letters'
export type { VaultRow, WelcomeAction } from './welcome/welcome'

export { default as Palette } from './palette/Palette.vue'
export { default as KeyCap } from './palette/KeyCap.vue'
export { commandKeyChord, keyChord } from './palette/item'
export type { ActionWords, PaletteItem, PaletteKeys, PaletteGroup } from './palette/item'

/** The size a node's label is being set at, and the plex drawn to hold it. */
export { optionsForType, useTypeSize } from './plex/sizing'

export { default as Editor } from './editor/Editor.vue'
/** A time against every line of an editor, and the line being said now. */
export { timing } from './editor/timing'

/** The controls a recording is played by. What plays is somewhere else. */
export { default as Player } from './player/Player.vue'
export { clock } from './player/clock'

/** A stream taken up again for as long as a window is open. */
export { following } from './stream/stream'

export { default as Spinner } from './waiting/Spinner.vue'
export { default as Skeleton } from './waiting/Skeleton.vue'

export type { Turn } from './thread/turn'
export { conversation } from './thread/conversation'
export type { Conversation } from './thread/conversation'
export type { AgentPort, AgentStep } from './thread/agent'

/** A link to a note: `[[name]]` in the text, `note://<identifier>` inside it. */
export { pointsAtNote, wikilinksIn } from './linking/address'

/** A link that leads out of the application, and the window held against it. */
export { holdsTheWindow } from './linking/outward'

/** Numbers as they are read out, which both windows read the same way. */
export { many, percent, plural } from './digits'

export { default as Prose } from './prose/Prose.vue'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Tone } from './activity/tally'

export { default as Notices } from './notices/Notices.vue'
export { noticed } from './notices/notice'
export type { Notice, Stay, Task } from './notices/notice'

export { default as Agent } from './screens/Agent.vue'
export { default as PageReader } from './reader/PageReader.vue'

export { default as WorkspaceLayout } from './workspace/WorkspaceLayout.vue'
export { closeTab, openTab, openTabBeside } from './workspace/edit'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './workspace/node'
export { paneById, panesOf } from './workspace/tree'
export type { Tab, Workspace } from './workspace/node'

export { default as Tree } from './tree/Tree.vue'
export type { Row, RowMarker } from './tree/row'

export { default as StencilEditor } from './cards/StencilEditor.vue'
export { default as DeckEditor } from './cards/DeckEditor.vue'
export { default as CardProse } from './cards/CardProse.vue'

/** The schemes a link in a card may point at. An address naming none is the caller's. */
export { scheme } from './cards/safe'
/** For putting a card or a field where a person let it go, without drawing it. */
export { ordered, reordered } from './cards/order'
export type { Half, InsertionPoint } from './cards/order'
/**
 * Where a card let go at the head of a deck lands, before its first section,
 * and where one let go past the last card standing under a heading lands.
 */
export { blanks as cardBlanks, ended as cardEnded, endOf as cardEndOf, HEAD as CARD_HEAD } from './cards/deck'
export { declared as cardFields } from './cards/order'
export type { DeckCard, DeckSection } from './cards/deck'
export type { Stencil } from './cards/stencil'

export { byHolding } from './plex/reaching'
export { byDoubleTap } from './plex/showing'
export type { PlexPart } from './plex/inside'
export type { EdgeArrow, PlexEdge } from './plex/edge'
export type { PlexNeighbourhood } from './plex/neighbourhood'
export type { PlexNode, Position } from './plex/node'
export type { PlexRelatedSeat } from './plex/seat'
export type { PlexShowing } from './plex/showing'
