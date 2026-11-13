/**
 * Props and events of the plex drawing: the edge layer between nodes, and one
 * node.
 */
import type { VNode } from 'vue'
import type { MenuTrigger } from '@numen/ui'
import type { PlexFrame } from '../../lib/frame'
import type { NodeHeadings } from '../../lib/inside'
import type { NamesRoom } from '../../lib/names'
import type { GestureRole, PlacedNode, Position } from '../../lib/node'
import type { WideBox } from '../../model/sizing/expansion'
import type { Attention } from '../../model/useAttention'
import type { PlexDragState } from '../../model/interaction/drag'
import type { Gesture } from '../../model/interaction/gesture'
import type { PlexBehavior } from '../../model/behavior'
import type { PlexRelation, PlexDestination } from '../../../types'

export interface PlexPending {
  from: string
  at: Position
}

/** `gesture` and `dragging` are passed as the refs they are, so the pointer moving through them redraws the lines it draws and not the notes. */
export interface PlexViewProps {
  frame: PlexFrame
  viewport: { width: number; height: number }
  nodeSize: { width: number; height: number }
  ghostSize?: { width: number; height: number } | undefined
  showEdgeLabels?: boolean
  showPills?: boolean
  highlightLinks?: boolean
  namesOpacity: number
  focusedNamesOpacity: number
  canReach?: boolean
  relationName?: (relation: PlexRelation) => string
  widen?: ((node: PlacedNode) => WideBox | null) | undefined
  hung?: ((node: PlacedNode) => NodeHeadings | null) | undefined
  roomOf?: ((id: string) => NamesRoom | undefined) | undefined
  behavior?: PlexBehavior
  gesture?: Gesture | undefined
  dragging?: PlexDragState | undefined
  pending?: PlexPending | null
  dropName?: (relation: PlexRelation) => string
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
  /** A name set around a node was chosen. The identifier is the caller's, returned unchanged. */
  (event: 'choose-name', id: string): void
  /** A menu or inspector was requested on an edge. */
  (event: 'edge-menu', pair: string, point: Position): void
}

/** `attention` is what the drawing attends to: a node stays on only while the attention is on no other node. */
export interface PlexNodeProps {
  node: PlacedNode
  gestureRole?: GestureRole
  wide?: WideBox | null
  hung?: NodeHeadings | null
  behavior?: PlexBehavior
  showPills?: boolean
  restHeight?: number
  viewportWidth?: number | undefined
  attention?: Attention
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
  /** The box began to open, or has closed or gone from the drawing. */
  (event: 'settle', resting: boolean): void
  /** The pointer or keyboard focus came onto the node, or left it. */
  (event: 'attend', isAttended: boolean): void
  /** A heading of this node was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}

export interface PlexDrawnSlots {
  /** Content drawn next to a node's title. */
  icon?(props: { node: PlacedNode }): VNode[]
}
