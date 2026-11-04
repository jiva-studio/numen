import type { Component } from 'vue'
import type { Position } from '@numen/ui'
import type { EligibleLinkTab, TabId, TabWindow } from '../types'

export interface TabLinkMenuProps {
  tabId: TabId
  eligibleTabs: readonly EligibleLinkTab[]
  at: Position
  getTabIcon: (tab: TabId) => Component | null
  /** Returns the icon of a kind; used for tabs of other windows and the active tab of a window. */
  getKindIcon?: (kind: string) => Component | null
  isLinked?: boolean
  linkedTargetTitle?: string
  /** Offers to open the tab in a new window. */
  canDetach?: boolean
  /** The other windows the tab can move to, the main window first. */
  windows?: readonly TabWindow[]
  /** Whether the tab is asking; an asking tab is not moved or detached. */
  isAsking?: boolean
}
