/**
 * The line a gesture drags behind it, and the box it would leave where it ends.
 *
 * A box drawn here is a node like any other, at the same size in the same
 * place — it is only that it has no name yet, and says the relation it would take
 * instead, in whatever words it was given.
 */
import { createThreadPath, type Drop, type Size } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import { ghostNode, getHandle, type PlacedNode, type Position } from '../../lib/node'
import type { PlexRelation } from '../../lib/relation'

type Thread = ReturnType<typeof createThreadPath>

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
  const offset = getHandle(source)
  return createThreadPath({ x: source.x + offset.x, y: source.y }, to)
}

/**
 * The line from a node's handle to a point given in the plex's own pixels,
 * measured from its top left corner.
 */
export function getPendingThread(
  frame: PlexFrame,
  from: string | null,
  at: Position | null,
  viewport: Size,
): Thread | null {
  const to = at && { x: at.x - viewport.width / 2, y: at.y - viewport.height / 2 }
  return getReachThread(frame, from, to)
}

/**
 * The node a gesture would make, drawn where it would appear so the reader sees
 * it before letting go.
 */
export function getReachGhost(
  outcome: Drop | null,
  to: Position | null,
  relationName: (relation: PlexRelation) => string,
  nodeSize: Size,
): PlacedNode | null {
  if (outcome?.kind !== 'create' || !to) return null
  return ghostNode('ghost', relationName(outcome.relation), outcome.relation, to, nodeSize)
}

/**
 * Something dragged over the picture: the line from the focus to the pointer,
 * and the shape letting go would leave there.
 *
 * Drawn only where letting go comes to a relation, so what the reader sees and what
 * the gesture answers are the one thing.
 */
export function getDraggedShape(
  frame: PlexFrame,
  relation: PlexRelation | null,
  to: Position | null,
  dropName: (relation: PlexRelation) => string,
  nodeSize: Size,
): DraggedShape | null {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!relation || !to || !focus) return null

  return {
    thread: createThreadPath(focus, to),
    ghost: ghostNode('dragged', dropName(relation), relation, to, nodeSize),
  }
}
