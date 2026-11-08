import type { Component } from 'vue'
import type { Position } from '@numen/ui'
import type { EligibleLinkTab, TabId, TabWindow } from '../types'

export interface TabLinkMenuProps {
  tabId: TabId
  eligibleTabs: readonly EligibleLinkTab[]
  at: Position
  getTabIcon: (tab: TabId) => Component | null
  getKindIcon?: (kind: string) => Component | null
  isLinked?: boolean
  linkedTargetTitle?: string
  canDetach?: boolean
  windows?: readonly TabWindow[]
  isAsking?: boolean
}
