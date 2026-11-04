import type { Component } from 'vue'
import type { Position } from '@numen/ui'
import type { EligibleLinkTab, TabId } from '../types'

export interface TabLinkMenuProps {
  tabId: TabId
  eligibleTabs: readonly EligibleLinkTab[]
  at: Position
  getTabIcon: (tab: TabId) => Component | null
  /** Returns the icon of a kind; used for tabs of other windows. */
  getKindIcon?: (kind: string) => Component | null
  isLinked?: boolean
  linkedTargetTitle?: string
  /** Offers to open the tab in a new window. */
  canDetach?: boolean
}
