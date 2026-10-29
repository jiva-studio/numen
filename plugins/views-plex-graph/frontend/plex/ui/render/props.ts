/**
 * Props and events of the plex drawing: the edge layer between nodes, and one
 * node.
 */
import type { MenuTrigger } from '@numen/ui'
import type { Drop } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import type { NodeHeadings } from '../../lib/inside'
import type { GestureRole, PlacedNode, Position } from '../../lib/node'
import type { PlexRelation } from '../../lib/relation'
import type { WideBox } from '../../model/sizing/expansion'
import type { PlexBehaviour } from '../../model/behaviour'
import type { PlexDestination } from '../../model/interaction/destination'

export interface PlexPending {
  from: string
  at: Position
}

export interface PlexViewProps {
  frame: PlexFrame
  /** The viewport to centre on, measured by the element's owner. */
  viewport: { width: number; height: number }
  /** How big a node the gesture would make, for the shape drawn under it. */
  nodeSize: { width: number; height: number }
  /** Draw the title a typed relationship carries. */
  showEdgeLabels?: boolean
  /** Whether reaching out is allowed at all, and so whether any node may
   *  offer a handle. */
  canReach?: boolean
  /**
   * Label of a relation, used where a relation is written into the drawing:
   * the outline of a gesture shows the relation it would create.
   */
  relationName?: (relation: PlexRelation) => string
  /**
   * The box a node expands to while it has focus, or null for a node whose
   * title is already fully shown. Text is measured where the plex is drawn, so
   * the box is passed in already computed.
   */
  widen?: ((node: PlacedNode) => WideBox | null) | undefined
  /**
   * The headings a node shows under its box while it has focus, or null for a
   * node with none. The drawing determines which headings a node has.
   */
  hung?: ((node: PlacedNode) => NodeHeadings | null) | undefined
  /** Pointer and animation behaviour of the nodes. Defaults to the standard behaviour. */
  behaviour?: PlexBehaviour
  /** A gesture in progress: where it started, where it is, what it means. */
  gestureFrom?: string | null
  gestureAt?: Position | null
  gestureOutcome?: Drop | null
  /**
   * A node awaiting a partner: the node a line starts from, and the point it
   * ends at, in plex pixels from the top-left corner.
   */
  pending?: PlexPending | null
  /**
   * An external drag over the drawing: the pointer position and the relation
   * created by dropping there. Both must be set, otherwise nothing is drawn.
   */
  draggedAt?: Position | null
  dropRelation?: PlexRelation | null
  /**
   * Label of the relation an external drop would create, written into the
   * drawing. English by default.
   */
  dropName?: (relation: PlexRelation) => string
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
  /** The role of this node in the current gesture; it cannot be derived from the node. */
  gestureRole?: GestureRole
  /**
   * The box the node expands to while it has focus, or null when its title is
   * already fully shown.
   */
  wide?: WideBox | null
  /**
   * The headings shown under the node's box while it has focus, or null for a
   * node with none.
   */
  hung?: NodeHeadings | null
  /** Pointer and animation behaviour of this node. Defaults to the standard behaviour. */
  behaviour?: PlexBehaviour
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
   * press has no pointer position, so the centre of the box is used.
   */
  (event: 'menu', point: Position, trigger: MenuTrigger): void
  /**
   * The node gained or lost focus. The expanded box is drawn last, and only
   * the whole drawing knows which box that is.
   */
  (event: 'settle', resting: boolean): void
  /** A heading of this node was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}

export interface PlexDrawnSlots {
  /** Content drawn next to a node's title. */
  icon?(props: { node: PlacedNode }): unknown
}
