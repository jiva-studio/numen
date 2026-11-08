import type { Component } from 'vue'
import type { Orientation, WorkspaceNode } from '../../../lib/node'

export interface BranchPanelProps {
  node: WorkspaceNode
  index: number
  share: number
  floor: number
  axis: Orientation
  depth: number
  direction: Orientation
  branchView: Component
}
