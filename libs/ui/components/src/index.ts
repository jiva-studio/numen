/**
 * What an application may use.
 */
import './shared/tokens/theme.css'
import './shared/tokens/window.css'

export { Menu, useOutsidePress } from './shared/ui/menu'
export type { MenuItem, MenuTrigger } from './shared/ui/menu'

export { Button } from './shared/ui/button'

export { QrCode } from './shared/ui/qr-code'

export { QrScanner } from './shared/ui/qr-scanner'
export { hasCamera, type ScannerCamera } from './shared/api/camera'

export { NumberField } from './shared/ui/number-field'

export { Switch } from './shared/ui/switch'

export { Slider } from './shared/ui/slider'

export { SegmentedControl } from './shared/ui/segmented'

export { Select } from './shared/ui/select'

export { SearchableList, SearchableListPopup } from './shared/ui/searchable-list'
export type { ListItem } from './shared/ui/searchable-list'

export { isTyping } from './shared/lib/dom/typing'

export { KeyCap, getKeyChord } from './shared/ui/key-cap'
export type { Keystroke } from './shared/ui/key-cap'
export { Fingerprint } from './shared/ui/fingerprint'

export {
  createRequestGuard,
  type InFlightRequest,
  type RequestGuard,
} from './shared/lib/async/requestGuard'

export { runInBackground } from './shared/lib/async/background'

export { reportFailure } from './shared/api/report'

export { Spinner } from './shared/ui/spinner'

export { StageList, type Stage, type StageState } from './shared/ui/stage-list'

export { Toast } from './shared/ui/toast'

export { createNamedIcon } from './shared/ui/named-icon'
export { LoadingIndicator } from './shared/ui/loading-indicator'
export { ErrorState } from './shared/ui/error-state'
export {
  CardAction,
  CardActions,
  IslandCard,
  type CardActionChoice,
  type CardActionItem,
} from './shared/ui/island-card'

export { ConfirmDialog } from './shared/ui/confirm-dialog'
export { Dialog, provideDialogHost } from './shared/ui/dialog'
export { Skeleton } from './shared/ui/skeleton'

export { groupDigits, formatCount, formatPercent, pickPlural } from './shared/lib/format/digits'

export { Prose } from './shared/ui/prose'
export type { LinkPiece } from './shared/lib/text/marks'

export type { TextRange } from './shared/lib/text/textRange'

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
export { joinClasses } from './shared/lib/dom/classes'

export { stepTo } from './shared/lib/dom/roving'

export { DragPreview, usePressDrag } from './shared/ui/drag-preview'

export { Tooltip, useTooltip } from './shared/ui/tooltip'
export { isPageMove, useMenuDismissal } from './shared/ui/menu/interaction/menuDismissal'

export { Tree } from './features/tree'
export type { Row, RowBadge, RowMarker } from './features/tree'
