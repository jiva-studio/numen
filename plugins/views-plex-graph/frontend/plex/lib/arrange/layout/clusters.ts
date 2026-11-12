/** Each relation grows as a compact cluster from its side of the focus, one note per relation in turn. */
import { RELATION_NAMES, RELATIONS, type Lean } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Direction, PlexOptions } from '../options'
import { findFreeSpot, SpotIndex, type Spot, type Vector } from './freeSpot'
import type { NodesByRelation, Placement, PlacementContext } from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** How far a leaning group tilts across its axis, as a fraction of the way it grows. */
const LEAN = 0.42

/** Notes one per relation in turn, so every relation gets a share of the near space. */
export function takeTurns(nodesByRelation: NodesByRelation): PlexNode[] {
  const queues = RELATION_NAMES.map((relation) => nodesByRelation[relation] ?? [])
  const turns: PlexNode[] = []
  const rounds = Math.max(0, ...queues.map((queue) => queue.length))
  for (let round = 0; round < rounds; round++) {
    for (const queue of queues) if (round < queue.length) turns.push(queue[round]!)
  }
  return turns
}

const SIDES: Readonly<Record<Direction, Vector>> = {
  up: { x: 0, y: -1 },
  down: { x: 0, y: 1 },
  left: { x: -1, y: 0 },
  right: { x: 1, y: 0 },
}

/** A unit vector: the side the group grows towards, tilted across the axis by its lean. */
export function getClusterDirection(side: Direction, lean: Lean | null): Vector {
  const { x, y } = SIDES[side]
  if (lean === null || x === 0) return { x, y }
  const tilt = lean === 'down' ? LEAN : -LEAN
  const length = Math.hypot(x, tilt)
  return { x: x / length, y: tilt / length }
}

/** The nearest point a group may reach: clear of the focus on every axis the group moves along. */
export function getAnchor(
  direction: Vector,
  focus: { readonly width: number; readonly height: number },
  options: PlexOptions,
): Vector {
  const { focusGap, lineGap } = options
  if (direction.x === 0) return { x: 0, y: direction.y * (focus.height / 2 + focusGap) }
  if (direction.y === 0) return { x: direction.x * (focus.width / 2 + focusGap), y: 0 }
  return {
    x: Math.sign(direction.x) * (focus.width / 2 + focusGap),
    y: Math.sign(direction.y) * (focus.height / 2 + lineGap),
  }
}

interface MiddleRequest {
  /** How many notes of the relation are placed. */
  readonly size: number
  readonly cell: { readonly width: number; readonly height: number }
}

/** The middle of a group sits as far out from its start as half the patch is deep. */
export function getClusterMiddle(start: Vector, direction: Vector, request: MiddleRequest): Vector {
  const { size, cell } = request
  const depth =
    Math.max(0, (Math.sqrt(size) - 1) / 2) *
    (Math.abs(direction.x) * cell.width + Math.abs(direction.y) * cell.height)
  return { x: start.x + direction.x * depth, y: start.y + direction.y * depth }
}

function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const { options, width, shouldStop } = context
  const gaps = { gap: options.gap, lineGap: options.lineGap }
  const { height } = options.nodeSize
  const taken = new SpotIndex([
    { x: focus.x, y: focus.y, width: focus.width, height: focus.height },
  ])
  const groups = new Map<PlexRelation, Spot[]>()
  const placed = new Map<string, PlacedNode>()

  for (const node of takeTurns(nodesByRelation)) {
    const relation = node.relation as PlexRelation
    const members = nodesByRelation[relation] ?? []
    const group = groups.get(relation) ?? []
    groups.set(relation, group)

    const size = { width: width(node), height }
    const direction = getClusterDirection(options.direction[relation], RELATIONS[relation].lean)
    const anchor = getAnchor(direction, focus, options)
    const reach =
      Math.abs(direction.x) * (size.width / 2) + Math.abs(direction.y) * (size.height / 2)
    const start = { x: anchor.x + direction.x * reach, y: anchor.y + direction.y * reach }
    const cell = { width: size.width + gaps.gap, height: size.height + gaps.lineGap }
    const middle = getClusterMiddle(start, direction, { size: members.length, cell })

    const spot = findFreeSpot(taken, { start, middle, direction, size, group, gaps })
    taken.add(spot)
    group.push(spot)
    const box: PlacedNode = {
      ...node,
      x: spot.x,
      y: spot.y,
      width: spot.width,
      height: spot.height,
      order: members.indexOf(node),
      opacity: 1,
    }
    placed.set(node.id, box)
    if (shouldStop?.(box)) break
  }

  return RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => placed.get(node.id) ?? []),
  )
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
