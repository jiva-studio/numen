/**
 * The line a gesture drags behind it, and the box it would leave where it ends.
 *
 * A box drawn here is a node like any other, at the same size in the same
 * place — it is only that it has no name yet, and says the seat it would take
 * instead, in whatever words it was given.
 */
import { threadOf, type Drop, type Size } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import { ghostNode, handleIn, type PlacedNode, type Position } from '../../lib/node'
import type { PlexRelatedSeat } from '../../lib/seat'

type Thread = ReturnType<typeof threadOf>

/** A line ending in the box letting go there would leave. */
export interface DraggedShape {
  readonly thread: Thread
  readonly ghost: PlacedNode
}

/** The line from a node's handle to the pointer, and nothing off a node. */
export function getReachThread(
  frame: PlexFrame,
  from: string | null,
  to: Position | null,
): Thread | null {
  const source = frame.nodes.find((node) => node.id === from)
  if (!source || !to) return null
  const offset = handleIn(source)
  return threadOf({ x: source.x + offset.x, y: source.y + offset.y }, to)
}

/**
 * The node a gesture would make, drawn where it would appear so the reader sees
 * it before letting go.
 */
export function getReachGhost(
  outcome: Drop | null,
  to: Position | null,
  seatName: (seat: PlexRelatedSeat) => string,
  nodeSize: Size,
): PlacedNode | null {
  if (outcome?.kind !== 'create' || !to) return null
  return ghostNode('ghost', seatName(outcome.seat), outcome.seat, to, nodeSize)
}

/**
 * Something dragged over the picture: the line from the focus to the pointer,
 * and the shape letting go would leave there.
 *
 * Drawn only where letting go comes to a seat, so what the reader sees and what
 * the gesture answers are the one thing.
 */
export function getDraggedShape(
  frame: PlexFrame,
  seat: PlexRelatedSeat | null,
  to: Position | null,
  dropName: (seat: PlexRelatedSeat) => string,
  nodeSize: Size,
): DraggedShape | null {
  const focus = frame.nodes.find((node) => node.seat === 'focus')
  if (!seat || !to || !focus) return null

  return {
    thread: threadOf(focus, to),
    ghost: ghostNode('dragged', dropName(seat), seat, to, nodeSize),
  }
}
