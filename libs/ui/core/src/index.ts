/**
 * What an application may use.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { Menu, groupItems, useOutsidePress } from './shared/ui/menu'
export type { MenuItem, MenuTrigger } from './shared/ui/menu'

export { Button } from './shared/ui/button'

/** A single line of digits typed by the user and clamped to its bounds. */
export { NumberField } from './shared/ui/number-field'

export { Switch } from './shared/ui/switch'

/** One value along a track, moved by a handle. */
export { Slider } from './shared/ui/slider'

/** Two to four choices side by side, one of them chosen. */
export { SegmentedControl } from './shared/ui/segmented'

/** One choice out of a list, taken from a menu the machine draws. */
export { Select } from './shared/ui/select'
export type { SelectChoice } from './shared/ui/select'

/** Whether a key was pressed into something being written in. */
export { isTyping } from './shared/lib/dom/typing'

export { KeyCap, keyChord } from './shared/ui/key-cap'
export type { PaletteIcon, PaletteKeys } from './shared/ui/key-cap'

/** A moment of a recording as a player writes one. */
export { clock } from './shared/lib/time/duration'

/** A stream taken up again for as long as a window is open. */
export { createFollower } from './shared/lib/async/stream'

/** Runs a promise without awaiting it and passes its rejection to a callback. */
export { runInBackground } from './shared/lib/async/background'

/** Rethrows a rejection of a background task as an uncaught error. */
export { reportFailure } from './shared/lib/async/report'

export { Spinner } from './shared/ui/spinner'

/** A Lucide icon drawn by the name a manifest gives it. */
export { NamedIcon, createNamedIcon } from './shared/ui/named-icon'
export { LoadingIndicator } from './shared/ui/loading-indicator'
export { Skeleton } from './shared/ui/skeleton'

/** A link that leads out of the application, and the window held against it. */
export { interceptExternalLinks, isExternalLink } from './shared/lib/text/externalLink'

/** Numbers as they are read out, which both windows read the same way. */
export { groupDigits, many, percent, plural } from './shared/lib/format/digits'

export { Prose } from './shared/ui/prose'
export type { LinkPiece } from './shared/lib/text/marks'

export type { TextRange } from './shared/lib/text/textRange'

/** The size, space and position a component outside this module uses for its layout. */
export type { Position, Size } from './shared/lib/math/geometry'
export { clamp } from './shared/lib/math/bounds'
export { getAdjacentPlace, type Box } from './shared/lib/dom/place'
export {
  browserViewport,
  useFreeSpace,
  useViewport,
  useWholeViewport,
  type Viewport,
} from './shared/lib/dom/viewport'
export { getPageNavigationAction, type PageNavigationAction } from './shared/lib/dom/pageNavigation'
export { browserClock, onNextFrame, type Clock, type Paint } from './shared/lib/time/clock'
export { createRefRegistry } from './shared/lib/dom/refs'
export { cn } from './shared/lib/dom/classes'

/** A row of a list the arrow keys walk, and a pair of UTF-16 units read as one character. */
export { stepTo } from './shared/lib/dom/roving'
export { isLeading, isTrailing } from './shared/lib/text/surrogate'

/** A thing carried under the pointer, and a menu put away by a press outside it or Escape. */
export { DragPreview, usePressDrag } from './shared/ui/drag-preview'
export { useMenuDismissal } from './shared/ui/menu/interaction/menuDismissal'

export { Tree } from './features/tree'
export type { Row, RowMarker } from './features/tree'
