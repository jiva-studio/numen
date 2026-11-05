import type { Clock } from '@numen/ui'
import type { NodeIdFactory } from '../lib/edit'
import type { WindowEdge } from '../lib/edges'
import type { Tab } from '../lib/node'

export interface WorkspaceLayoutProps {
  tabs: readonly Tab[]
  /** Where identities for what a gesture makes come from. */
  createId?: NodeIdFactory | undefined
  /** How close to the outer edge divides the whole workspace. */
  edge?: number
  /** Distance the pointer moves before a press becomes a drag. */
  threshold?: number
  /** The minimum size at which a pane is rendered. */
  minimum?: number
  /** The clock. Browser by default; a test hands in its own. */
  clock?: Clock
  /**
   * Given while the workspace is the top of the window, with no title bar above it: the strips of
   * the panes at the top edge move the window by their free space, and the strips in the top
   * corners keep the room given free for the window buttons.
   */
  windowEdge?: WindowEdge | undefined
}
