/**
 * What the plex is told, and what it says back.
 */
import type { MenuOpening } from '@/shared/ui/menu'
import type { Viewport } from '@/shared/lib/viewport'
import type { Placement, PlexOptionsInput } from '../lib/arrange'
import type { PlexPart } from '../lib/inside'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlacedNode, Position } from '../lib/node'
import type { PlexRelatedSeat } from '../lib/seat'
import type { ReachStrategy } from '../model/reaching'
import type { PlexDestination, ShowStrategy } from '../model/showing'
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
   * How much room the plex has, and what it becomes. Browser by default; a
   * test hands in its own and every coordinate is then a value it can name.
   */
  viewport?: Viewport
  /**
   * Seats a gesture may produce. A sibling is another of the parent's
   * children, so it is left out; which relationships exist is the caller's
   * to say.
   */
  creatable?: readonly PlexRelatedSeat[]
  /** How far a gesture travels before it is a drag and not a click. */
  dragThreshold?: number
  /**
   * How long the attention rests on a box before it widens to the whole of
   * its title. Milliseconds; nothing at all never widens.
   */
  dwell?: number
  /** How a node offers to be reached out of. The handle by default. */
  reaching?: ReachStrategy
  /** How a node is asked for on its own. The second click by default. */
  showing?: ShowStrategy
  /**
   * The parts of a node, asked for by the node's own identifier. They come
   * out from under its box while the attention rests on it, and a node named
   * none for hangs nothing.
   */
  parts?: (id: string) => readonly PlexPart[]
  /**
   * What to call a seat, for the outline a gesture draws and for the
   * overflow line. English by default.
   */
  seatName?: (seat: PlexRelatedSeat) => string
  /**
   * What is being dragged over the picture from somewhere else. Each
   * identifier is opaque and all of them are handed back untouched; an empty
   * list is nothing dragged, and the picture then draws none of it.
   */
  dragged?: readonly string[]
  /**
   * What to call what letting go with something dragged in would do. English
   * by default.
   */
  dropName?: (seat: PlexRelatedSeat) => string
}

export interface PlexEvents {
  /** A node other than the focus was chosen, by click or by keyboard. */
  (event: 'activate', id: string): void
  /**
   * A node asked for on its own: a double click, or a press with Shift held.
   * Where it is to be drawn is the second word, and the focus answers this as
   * every other node does.
   */
  (event: 'show', id: string, showing: PlexDestination): void
  /** Reached out into empty space: make a node in this seat of that one. */
  (event: 'create', from: string, seat: PlexRelatedSeat): void
  /** Reached out onto another node: relate the two in this seat. */
  (event: 'link', from: string, to: string, seat: PlexRelatedSeat): void
  /**
   * What was dragged in from outside was let go over the picture: relate each
   * of them to the focus in this seat. The identifiers are the ones they were
   * handed in as.
   */
  (event: 'bring', dragged: readonly string[], seat: PlexRelatedSeat): void
  /**
   * A menu was asked for on a node: which node, where on the screen, and what
   * asked for it. A keypress carries no point, so the middle of the box is
   * where it is asked.
   *
   * Every node answers this, the focus included. What the menu holds and what
   * choosing an item does are the caller's.
   */
  (event: 'menu', id: string, at: Position, opening: MenuOpening): void
  /** A menu asked for on a node has nothing left to stand on. */
  (event: 'dismiss'): void
  /**
   * A part of a node was chosen. Both identifiers are the caller's, handed
   * back as given.
   */
  (event: 'enter', id: string, part: string): void
}

export interface PlexSlots {
  /** What is drawn beside a node's title. A node with none is drawn narrower. */
  icon?(props: { node: PlacedNode }): unknown
  /** What is said about the neighbours that did not fit, in the caller's words. */
  overflow?(props: { overflow: readonly [PlexRelatedSeat, number][] }): unknown
}
