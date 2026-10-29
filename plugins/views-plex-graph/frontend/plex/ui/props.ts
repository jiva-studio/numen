/**
 * Props and events of the plex.
 */
import type { MenuTrigger, Viewport } from '@numen/ui'
import type { Placement, PlexOptionsInput } from '../lib/arrange'
import type { PlexHeading } from '../lib/inside'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlacedNode, Position } from '../lib/node'
import type { PlexRelation } from '../lib/relation'
import type { PlexPending } from './render/props'
import type { ReachStrategy } from '../model/interaction/reachStrategy'
import type { PlexDestination, ShowStrategy } from '../model/interaction/destination'
import type { Clock } from '../model/transition'

export interface PlexProps {
  neighbourhood: PlexNeighbourhood
  options?: PlexOptionsInput
  /** Rows and columns unless another arrangement is handed in. */
  placement?: Placement
  showEdgeLabels?: boolean
  /** Milliseconds. Zero arrives instantly. */
  duration?: number
  /** The clock. Browser by default; a test hands in its own. */
  clock?: Clock
  /**
   * Viewport size and its changes. Browser by default; a test supplies its own,
   * which makes every coordinate a known value.
   */
  viewport?: Viewport
  /**
   * Relations a gesture may produce. A sibling is another of the parent's
   * children, so it is left out; which relationships exist is the caller's
   * to say.
   */
  creatable?: readonly PlexRelation[]
  /** Distance in pixels a pointer moves before a gesture is a drag and not a click. */
  dragThreshold?: number
  /**
   * How long the attention rests on a box before it widens to the whole of
   * its title. Milliseconds; nothing at all never widens.
   */
  expansionDelay?: number
  /** How a node offers to be reached out of. The handle by default. */
  reaching?: ReachStrategy
  /** How a node is requested on its own. The second click by default. */
  showing?: ShowStrategy
  /**
   * The headings of a node, looked up by the node's identifier. They appear
   * below its box while the pointer rests on it. A node with no headings shows
   * none.
   */
  headings?: (id: string) => readonly PlexHeading[]
  /**
   * What to call a relation, for the outline a gesture draws and for the
   * overflow line. English by default.
   */
  relationName?: (relation: PlexRelation) => string
  /**
   * A node awaiting a partner, drawn as a line from the node to a point in the
   * plex's own pixels. Nothing draws no line.
   */
  pending?: PlexPending | null
  /**
   * Identifiers being dragged over the plex from outside. Each identifier is
   * opaque and is returned unchanged in the `bring` event. An empty list means
   * nothing is dragged, and the plex draws nothing for it.
   */
  dragged?: readonly string[]
  /**
   * What to call what letting go with something dragged in would do. English
   * by default.
   */
  dropName?: (relation: PlexRelation) => string
}

export interface PlexEvents {
  /** A node other than the focus was chosen, by click or by keyboard. */
  (event: 'activate', id: string): void
  /**
   * A node requested on its own: a double click, or a press with Shift held.
   * The second argument is the destination to draw it in, and the focus handles
   * this event like any other node.
   */
  (event: 'show', id: string, showing: PlexDestination): void
  /** Reached out into empty space: make a node in this relation of that one. */
  (event: 'create', from: string, relation: PlexRelation): void
  /** Reached out onto another node: relate the two in this relation. */
  (event: 'link', from: string, to: string, relation: PlexRelation): void
  /**
   * Identifiers dragged in from outside were dropped on the plex: relate each
   * of them to the focus in this relation. The identifiers are those supplied
   * in `dragged`.
   */
  (event: 'bring', dragged: readonly string[], relation: PlexRelation): void
  /**
   * A menu was requested on a node: the node, the screen position, and the
   * trigger. A keypress has no point, so the center of the box is used.
   *
   * Every node emits this, the focus included. The menu content and the effect
   * of choosing an item belong to the caller.
   */
  (event: 'menu', id: string, point: Position, trigger: MenuTrigger): void
  /** The menu requested on a node is closed. */
  (event: 'dismiss'): void
  /**
   * A heading of a node was chosen. Both identifiers are the caller's, handed
   * back as given.
   */
  (event: 'enter', id: string, heading: string): void
  /** A menu or inspector was requested on an edge between two nodes. */
  (event: 'edge-menu', pair: string, point: Position): void
}

export interface PlexSlots {
  /** What is drawn beside a node's title. A node with none is drawn narrower. */
  icon?(props: { node: PlacedNode }): unknown
  /** Content for the neighbours that did not fit, supplied by the caller. */
  overflow?(props: { overflow: readonly [PlexRelation, number][] }): unknown
}
