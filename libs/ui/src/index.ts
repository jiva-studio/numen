/**
 * What an application may use.
 *
 * Narrower than what the module contains: the easing curve, the routing and
 * the arithmetic are how a plex is built, not how it is used.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { Plex } from './features/plex'
export { getRelationName } from './features/plex'

export { Menu, groupItems, useOutsidePress } from './shared/ui/menu'
export type { MenuItem, MenuTrigger } from './shared/ui/menu'

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
export type { SegmentedChoice } from './shared/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './shared/ui/select'
export type { SelectChoice } from './shared/ui/select'

/** A day of the calendar, written down, read back and counted against another. */
export { dayAfter, parseDay, daysBetween, getDayName, isDay } from './shared/lib/time/day'
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
export { createTimeGutter } from './features/editor'
export { createWikilinkCompletion } from './features/editor'
export type { WikilinkOption, WikilinkCompletionOptions, Extension } from './features/editor'

/** A moment of a recording as a player writes one. */
export { clock } from './shared/lib/time/duration'

/** A stream taken up again for as long as a window is open. */
export { createFollower } from './shared/lib/async/stream'

export { Spinner } from './shared/ui/spinner'
export { LoadingOverlay, Waiting } from './shared/ui/waiting'
export { Skeleton } from './shared/ui/skeleton'

export type { Turn } from './features/thread'
export { useConversation } from './features/thread'
export type { Conversation } from './features/thread'
export type { AgentPort, AgentQuery, AgentStep } from './features/thread'

/** A link to a note: `[[name]]` in the text, `note://<identifier>` inside it. */
export { isNoteAddress, getWikilinks } from './shared/lib/text/address'

/** A link that leads out of the application, and the window held against it. */
export { interceptExternalLinks, isExternalLink } from './shared/lib/text/externalLink'

/** Numbers as they are read out, which both windows read the same way. */
export { many, percent, plural } from './shared/lib/format/digits'

export { Prose } from './shared/ui/prose'

/** What a count counts, and how a notice about it reads. */
export type { TallyUnit, Tone } from './features/notices'

export { Notices } from './features/notices'
export { createNotice } from './features/notices'
export type { Notice, Stay, Task } from './features/notices'

export { Agent } from './screens/agent'
export { Reader } from './features/reader'

export type { Span } from './shared/lib/text/span'

/** The size, room and placement a component outside this module lays itself out by. */
export type { Size } from './shared/lib/math/geometry'
export { clamp } from './shared/lib/math/bounds'
export { getAdjacentPlace, type Box } from './shared/lib/dom/place'
export { browserViewport, useRoom, useViewport, type Viewport } from './shared/lib/dom/viewport'
export { getPageNavigationAction, type PageNavigationAction } from './shared/lib/dom/pageNavigation'
export { onNextFrame } from './shared/lib/time/clock'
export { cn } from './shared/lib/dom/classes'

export { WorkspaceLayout } from './features/workspace'
export { closeTab, openTab, openTabBeside, type OpenBesidePlacement } from './features/workspace'

/** For arranging without drawing, or reading a gesture without this renderer. */
export { branch, pane } from './features/workspace'
export { paneById, getPanes } from './features/workspace'
export type { Tab, Workspace } from './features/workspace'

export { Tree } from './features/tree'
export type { Row, RowMarker } from './features/tree'

export { byLongPress } from './features/plex'
export { byDoubleTap } from './features/plex'
export type { PlexHeading, PlexPending } from './features/plex'
export type { EdgeArrow, PlexEdge } from './features/plex'
export type { PlexNeighbourhood } from './features/plex'
export type { PlexNode, Position } from './features/plex'
export type { PlexRelation } from './features/plex'
export type { PlexDestination } from './features/plex'
export { createThreadPath } from './features/plex'

export type { AgentModelOption, ReasoningLevelOption } from './features/message-composer'

export { StatusBar } from './features/status-bar'
export type { StatusBarTask, TaskState } from './features/status-bar'
