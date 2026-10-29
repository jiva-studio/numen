import type { Component } from 'vue'
import type { Orientation, WorkspaceNode } from '../../../lib/node'

export interface BranchPanelProps {
  node: WorkspaceNode
  /** The child's position among its siblings; the first child gets no handle. */
  index: number
  /** The share of the branch the panel opens at, in percent. */
  share: number
  /** The smallest share a handle may leave it, in percent. */
  floor: number
  axis: Orientation
  depth: number
  /** Which way the branch divides its length. */
  direction: Orientation
  /** What a child with children of its own is drawn with. */
  branchView: Component
}
