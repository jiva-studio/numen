import type { Component } from 'vue'
import type { Position } from '@numen/ui'
import type { EligibleLinkTab, TabId } from '../types'

export interface TabLinkMenuProps {
  tabId: TabId
  eligibleTabs: readonly EligibleLinkTab[]
  at: Position
  getTabIcon: (tab: TabId) => Component | null
  isLinked?: boolean
  linkedTargetTitle?: string
  /** Offers to open the tab in a new window. */
  canDetach?: boolean
}
