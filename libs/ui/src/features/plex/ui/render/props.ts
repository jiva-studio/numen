/**
 * What the drawing is told, and what it says back: the picture between the
 * nodes, and one node inside it.
 */
import type { MenuTrigger } from '@/shared/ui/menu'
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
  /** The window to centre on. Measured by whoever owns the element. */
  viewport: { width: number; height: number }
  /** How big a node the gesture would make, for the shape drawn under it. */
  nodeSize: { width: number; height: number }
  /** Draw the title a typed relationship carries. */
  showEdgeLabels?: boolean
  /** Whether reaching out is allowed at all, and so whether any node may
   *  offer a handle. */
  canReach?: boolean
  /**
   * What to call a relation, for the one place a relation has to be written into the
   * picture: the outline a gesture draws says which one it would take.
   */
  relationName?: (relation: PlexRelation) => string
  /**
   * The box a node widens to while the attention rests on it, and nothing
   * for a node with no more of its title to show. Text is measured where the
   * plex is drawn, so this arrives already worked out.
   */
  widen?: ((node: PlacedNode) => WideBox | null) | undefined
  /**
   * The headings a node hangs under its box while the attention rests on it,
   * and nothing for a node with none. Which headings a node holds is the
   * picture's to work out.
   */
  hung?: ((node: PlacedNode) => NodeHeadings | null) | undefined
  /** How the nodes behave under a hand and over time. The usual by default. */
  behaviour?: PlexBehaviour
  /** A gesture in progress: where it started, where it is, what it means. */
  gestureFrom?: string | null
  gestureAt?: Position | null
  gestureOutcome?: Drop | null
  /**
   * A node awaiting a partner: the node a line leaves, and the point it ends
   * at, in the plex's own pixels from its top left corner.
   */
  pending?: PlexPending | null
  /**
   * Something dragged over the picture from outside it: where the pointer
   * is, and the relation letting go there evaluates to. Both, or the picture draws
   * none of it.
   */
  draggedAt?: Position | null
  dropRelation?: PlexRelation | null
  /**
   * What to call what letting go with something dragged in would do, for the
   * one place it is written into the picture. English by default.
   */
  dropName?: (relation: PlexRelation) => string
}

export interface PlexViewEvents {
  /** A node was chosen. The identifier is the caller's, handed back as given. */
  (event: 'activate', id: string): void
  /** A node was asked for on its own, and where it is to be drawn. */
  (event: 'show', id: string, showing: PlexDestination): void
  /** A gesture began at a node's handle. */
  (event: 'reach', id: string, pointer: PointerEvent): void
  /** A handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask', id: string): void
  /** A menu was asked for on a node: which, where, and by what. */
  (event: 'menu', id: string, point: Position, trigger: MenuTrigger): void
  /** A heading of a node was chosen. Both identifiers are the caller's. */
  (event: 'enter', id: string, heading: string): void
  /** A menu or inspector was asked for on an edge. */
  (event: 'edge-menu', pair: string, point: Position): void
}

export interface PlexNodeProps {
  node: PlacedNode
  /** What this node is to the gesture. The one thing it cannot work out. */
  gestureRole?: GestureRole
  /**
   * The box it widens to while the attention rests on it, and nothing where
   * it has no more of its title to show.
   */
  wide?: WideBox | null
  /**
   * The headings it hangs under its box while the attention rests, and nothing
   * for a node with none.
   */
  hung?: NodeHeadings | null
  /** How this node behaves under a hand and over time. The usual by default. */
  behaviour?: PlexBehaviour
}

export interface PlexNodeEvents {
  /** Chosen, by click or by keyboard. Which node it was is the caller's to say. */
  (event: 'activate'): void
  /**
   * Asked to be drawn out on its own, and where it is to go. The modifier is
   * read here, so what travels on is the meaning.
   */
  (event: 'show', showing: PlexDestination): void
  /** A gesture began at the handle, and a pointer is dragging it somewhere. */
  (event: 'reach', pointer: PointerEvent): void
  /** The handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask'): void
  /**
   * A menu was asked for on this node: where it was asked, and what asked for
   * it. A keypress carries no point of its own, so the middle of the box is
   * where it is asked.
   */
  (event: 'menu', point: Position, trigger: MenuTrigger): void
  /**
   * The attention has settled on this node, or has left it. A widened box is
   * drawn last of all, and which box that is only the whole picture knows.
   */
  (event: 'settle', resting: boolean): void
  /** A heading of this node was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}

export interface PlexDrawnSlots {
  /** What is drawn beside a node's title. */
  icon?(props: { node: PlacedNode }): unknown
}
