/**
 * Props and events of the plex.
 */
import type { VNode } from 'vue'
import type { MenuTrigger, Viewport } from '@numen/ui'
import type { Placement, PlexOptionsInput } from '../lib/arrange'
import type { PlacedNode, Position } from '../lib/node'
import type { PlexPending } from './render/props'
import type { ReachStrategy } from '../model/interaction/reachStrategy'
import type { ShowStrategy } from '../model/interaction/destination'
import type { Clock } from '../model/transition'
import type { PlexType, Probe } from '../model/sizing/measure'
import type { PlexHeading, PlexGraph, PlexRelation, PlexDestination } from '../../types'

/**
 * With `showPills` off each title is drawn without its pill, after a dot in the hue of its
 * relation. `lineWidth` and `outlineWidth` are in pixels.
 */
export interface PlexProps {
  graph: PlexGraph
  options?: PlexOptionsInput
  placement?: Placement
  showEdgeLabels?: boolean
  showPills?: boolean
  lineWidth?: number
  outlineWidth?: number
  duration?: number
  clock?: Clock
  viewport?: Viewport
  creatable?: readonly PlexRelation[]
  dragThreshold?: number
  expansionDelay?: number
  reaching?: ReachStrategy
  showing?: ShowStrategy
  headings?: (id: string) => readonly PlexHeading[]
  relationName?: (relation: PlexRelation) => string
  pending?: PlexPending | null
  dragged?: readonly string[]
  dropName?: (relation: PlexRelation) => string
  readType?: (probe: Probe) => PlexType
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
  icon?(props: { node: PlacedNode }): VNode[]
  /** Content for the neighbors that did not fit, supplied by the caller. */
  overflow?(props: { overflow: readonly [PlexRelation, number][] }): VNode[]
}
