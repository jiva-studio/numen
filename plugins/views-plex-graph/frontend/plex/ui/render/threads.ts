/**
 * The line a gesture drags behind it, and the box it would create where it ends.
 *
 * A box drawn here is a node like any other, at the same size and place, except
 * that it has no name yet and shows the relation it would take, in the words it
 * was given.
 */
import { createThreadPath, type Drop, type Size } from '../../lib/arrange'
import type { PlexFrame } from '../../lib/frame'
import { ghostNode, getHandle, type PlacedNode, type Position } from '../../lib/node'
import type { PlexRelation } from '../../../types'

type Thread = ReturnType<typeof createThreadPath>

/** A line ending in the box dropping there would leave. */
export interface DraggedShape {
  readonly thread: Thread
  readonly ghost: PlacedNode
}

/** What decides where a thread starts: the handle of the node, which depends on its pill and on the width of the window. */
export interface ThreadOptions {
  readonly curvature?: number | undefined
  readonly hasPill?: boolean | undefined
  readonly viewportWidth?: number | undefined
}

/** The line from a node's handle to the pointer, and nothing off a node. */
export function getReachThread(
  frame: PlexFrame,
  from: string | null,
  to: Position | null,
  options: ThreadOptions = {},
): Thread | null {
  const source = frame.nodes.find((node) => node.id === from)
  if (!source || !to) return null
  const offset = getHandle(source, options.hasPill ?? true, options.viewportWidth)
  return createThreadPath({ x: source.x + offset.x, y: source.y }, to, options.curvature)
}

export interface PendingThreadTarget extends ThreadOptions {
  readonly from: string | null
  readonly position: Position | null
  readonly viewport: Size
}

/**
 * The line from a node's handle to a point given in the plex's own pixels,
 * measured from its top left corner.
 */
export function getPendingThread(frame: PlexFrame, target: PendingThreadTarget): Thread | null {
  const { from, position, viewport, ...options } = target
  const to = position && { x: position.x - viewport.width / 2, y: position.y - viewport.height / 2 }
  return getReachThread(frame, from, to, options)
}

export interface ReachGhostOptions {
  readonly relationName: (relation: PlexRelation) => string
  readonly nodeSize: Size
}

/**
 * The node a gesture would make, drawn where it would appear so the reader sees
 * it before dropping.
 */
export function getReachGhost(
  outcome: Drop | null,
  to: Position | null,
  options: ReachGhostOptions,
): PlacedNode | null {
  if (outcome?.kind !== 'create' || !to) return null
  return ghostNode(
    { id: 'ghost', title: options.relationName(outcome.relation), relation: outcome.relation },
    to,
    options.nodeSize,
  )
}

export interface DraggedShapeOptions {
  readonly dropName: (relation: PlexRelation) => string
  readonly nodeSize: Size
  readonly curvature?: number | undefined
}

/**
 * An item dragged over the plex: the line from the focus to the pointer, and
 * the shape that dropping would create there.
 *
 * Drawn only where the drop evaluates to a relation, so what the user sees
 * matches what the gesture does.
 */
export function getDraggedShape(
  frame: PlexFrame,
  target: { readonly relation: PlexRelation | null; readonly to: Position | null },
  options: DraggedShapeOptions,
): DraggedShape | null {
  const { relation, to } = target
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!relation || !to || !focus) return null

  return {
    thread: createThreadPath(focus, to, options.curvature),
    ghost: ghostNode(
      { id: 'dragged', title: options.dropName(relation), relation },
      to,
      options.nodeSize,
    ),
  }
}
