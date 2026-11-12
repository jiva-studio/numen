/**
 * Props and events of the plex drawing: the edge layer between nodes, and one
 * node.
 */
import type { VNode } from 'vue'
import type { MenuTrigger } from '@numen/ui'
import type { Drop } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import type { NodeHeadings } from '../../lib/inside'
import type { GestureRole, PlacedNode, Position } from '../../lib/node'
import type { WideBox } from '../../model/sizing/expansion'
import type { PlexBehavior } from '../../model/behavior'
import type { PlexRelation, PlexDestination } from '../../../types'

export interface PlexPending {
  from: string
  at: Position
}

export interface PlexViewProps {
  frame: PlexFrame
  viewport: { width: number; height: number }
  nodeSize: { width: number; height: number }
  showEdgeLabels?: boolean
  showPills?: boolean
  canReach?: boolean
  relationName?: (relation: PlexRelation) => string
  widen?: ((node: PlacedNode) => WideBox | null) | undefined
  hung?: ((node: PlacedNode) => NodeHeadings | null) | undefined
  behavior?: PlexBehavior
  gestureFrom?: string | null
  gestureAt?: Position | null
  gestureOutcome?: Drop | null
  pending?: PlexPending | null
  draggedAt?: Position | null
  dropRelation?: PlexRelation | null
  dropName?: (relation: PlexRelation) => string
  /** How far a line bows, as the routing setting; the lines being dragged bow by the same amount. */
  curvature?: number | undefined
}

export interface PlexViewEvents {
  /** A node was chosen. The identifier is the caller's, returned unchanged. */
  (event: 'activate', id: string): void
  /** A node was requested to open on its own, with its destination. */
  (event: 'show', id: string, showing: PlexDestination): void
  /** A gesture began at a node's handle. */
  (event: 'reach', id: string, pointer: PointerEvent): void
  /** A handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask', id: string): void
  /** A menu was requested on a node, with its position and trigger. */
  (event: 'menu', id: string, point: Position, trigger: MenuTrigger): void
  /** A heading of a node was chosen. Both identifiers are the caller's. */
  (event: 'enter', id: string, heading: string): void
  /** A menu or inspector was requested on an edge. */
  (event: 'edge-menu', pair: string, point: Position): void
}

export interface PlexNodeProps {
  node: PlacedNode
  gestureRole?: GestureRole
  wide?: WideBox | null
  hung?: NodeHeadings | null
  behavior?: PlexBehavior
  showPills?: boolean
}

export interface PlexNodeEvents {
  /** The node was chosen by click or keyboard. The caller identifies the node. */
  (event: 'activate'): void
  /**
   * The node was requested to open on its own, with its destination. The
   * modifier key is interpreted here, so the event carries the resulting
   * destination.
   */
  (event: 'show', showing: PlexDestination): void
  /** A gesture began at the handle, and the pointer is dragging it. */
  (event: 'reach', pointer: PointerEvent): void
  /** The handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask'): void
  /**
   * A menu was requested on this node, with its position and trigger. A key
   * press has no pointer position, so the center of the box is used.
   */
  (event: 'menu', point: Position, trigger: MenuTrigger): void
  /**
   * The node gained or lost focus. The expanded box is drawn last, and only
   * only the whole drawing determines which box that is.
   */
  (event: 'settle', resting: boolean): void
  /** A heading of this node was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}

export interface PlexDrawnSlots {
  /** Content drawn next to a node's title. */
  icon?(props: { node: PlacedNode }): VNode[]
}
