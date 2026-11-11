/**
 * What an application may use.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { Menu, useOutsidePress } from './shared/ui/menu'
export type { MenuItem, MenuTrigger } from './shared/ui/menu'

export { Button } from './shared/ui/button'

/** A single line of digits typed by the user and clamped to its bounds. */
export { NumberField } from './shared/ui/number-field'

export { Switch } from './shared/ui/switch'

/** A box that is ticked or not, with an optional label beside it. */

/** One value along a track, moved by a handle. */
export { Slider } from './shared/ui/slider'

/** Two to four choices side by side, one of them chosen. */
export { SegmentedControl } from './shared/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './shared/ui/select'

/** One search line over a list of rows in one frame, drawn in the window or as a popup. */
export { SearchableList, SearchableListPopup } from './shared/ui/searchable-list'
export type { ListItem } from './shared/ui/searchable-list'

/** Whether a key was pressed into something being written in. */
export { isTyping } from './shared/lib/dom/typing'

export { KeyCap, keyChord } from './shared/ui/key-cap'
export type { Keystroke } from './shared/ui/key-cap'

/** A moment of a recording as a player writes one. */
export { clock } from './shared/lib/time/duration'

/** A stream taken up again for as long as a window is open. */
export { createFollower } from './shared/lib/async/stream'
export {
  createRequestGuard,
  type InFlightRequest,
  type RequestGuard,
} from './shared/lib/async/requestGuard'

/** The order of the reads and writes of a store held by another process. */
export { createWriteOrder, type WriteOrder } from './shared/lib/async/remoteValues'

/** Runs a promise without awaiting it and passes its rejection to a callback. */
export { runInBackground } from './shared/lib/async/background'

/** Rethrows a rejection of a background task as an uncaught error. */
export { reportFailure } from './shared/api/report'
export type { Caught, Unvalidated } from './shared/lib/async/types'

export { Spinner } from './shared/ui/spinner'

/** A Lucide icon drawn by the name a manifest gives it. */
export { createNamedIcon } from './shared/ui/named-icon'
export { LoadingIndicator } from './shared/ui/loading-indicator'
export { Skeleton } from './shared/ui/skeleton'

/** A link that leads out of the application, and the window held against it. */
export { interceptExternalLinks } from './shared/api/externalLink'
export { isExternalLink } from './shared/lib/text/externalLink'

/** Numbers as they are read out, which both windows read the same way. */
export { groupDigits, many, percent, plural } from './shared/lib/format/digits'

export { Prose } from './shared/ui/prose'
export type { LinkPiece } from './shared/lib/text/marks'

export type { TextRange } from './shared/lib/text/textRange'

/** The size, space and position a component outside this module uses for its layout. */
export type { Position, Size } from './shared/lib/math/geometry'
export { clamp } from './shared/lib/math/bounds'
export { getAdjacentPlace, type Box } from './shared/lib/dom/place'
export { browserViewport, type Viewport } from './shared/api/viewport'
export { useFreeSpace, useViewport, useWholeViewport } from './shared/model/viewport'
export { getPageNavigationAction, type PageNavigationAction } from './shared/lib/dom/pageNavigation'
export { browserClock, getMonotonicNow, onNextFrame } from './shared/api/clock'
export type { Clock, Paint } from './shared/lib/time/clock'
export {
  doc,
  getCurrentDate,
  getNow,
  hasDocument,
  hasWindow,
  measureRect,
  measureWindow,
  nav,
  win,
} from './shared/api/dom'
export { createRefRegistry } from './shared/model/refs'
export { cn } from './shared/lib/dom/classes'

/** A row of a list the arrow keys walk, and a pair of UTF-16 units read as one character. */
export { stepTo } from './shared/lib/dom/roving'
export { isLeading, isTrailing } from './shared/lib/text/surrogate'

/** A thing carried under the pointer, and a menu put away by a press outside it or Escape. */
export { DragPreview, usePressDrag } from './shared/ui/drag-preview'
export { useMenuDismissal } from './shared/ui/menu/interaction/menuDismissal'


export { Tree } from './features/tree'
export type { Row, RowMarker } from './features/tree'
