import type { Clock, MenuItem, Position } from '@numen/ui'
import type { NodeIdFactory } from '../lib/edit'
import type { WindowEdge } from '../lib/edges'
import type { Tab, TabId } from '../lib/node'

/** The menu of a tab: the tab it was requested on, where, and the rows the caller gives it. */
export interface TabMenu {
  readonly tab: TabId
  readonly at: Position
  readonly items: readonly MenuItem[]
}

export interface WorkspaceLayoutProps {
  tabs: readonly Tab[]
  createId?: NodeIdFactory | undefined
  edge?: number
  threshold?: number
  minimum?: number
  clock?: Clock
  windowEdge?: WindowEdge | undefined
  tabMenu?: TabMenu | null | undefined
}
