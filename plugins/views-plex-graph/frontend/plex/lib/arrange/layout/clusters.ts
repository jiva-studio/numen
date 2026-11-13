/** Each relation grows as a compact cluster from its side of the focus, one note per relation in turn. */
import { RELATION_NAMES, RELATIONS, type Lean } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Direction, PlexOptions } from '../options'
import { findFreeSpot, SpotIndex, type Spot, type Vector } from './freeSpot'
import type { Size } from '../options'
import {
  getFootprint,
  getPillCenterY,
  type NodesByRelation,
  type Placement,
  type PlacementContext,
} from './placement'
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

/** What every note of one relation shares: the way the group grows, where it begins, and how far its first note reaches. */
interface GroupPlan {
  readonly members: readonly PlexNode[]
  readonly direction: Vector
  readonly anchor: Vector
  readonly firstReach: number
}

const getReach = (direction: Vector, box: { width: number; height: number }): number =>
  Math.abs(direction.x) * (box.width / 2) + Math.abs(direction.y) * (box.height / 2)

function createGroupPlans(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
  footprintOf: (node: PlexNode) => Size,
): ReadonlyMap<PlexRelation, GroupPlan> {
  const plans = new Map<PlexRelation, GroupPlan>()
  for (const relation of RELATION_NAMES) {
    const members = nodesByRelation[relation] ?? []
    const first = members[0]
    if (!first) continue
    const direction = getClusterDirection(
      context.options.direction[relation],
      RELATIONS[relation].lean,
    )
    plans.set(relation, {
      members,
      direction,
      anchor: getAnchor(direction, focus, context.options),
      firstReach: getReach(direction, footprintOf(first)),
    })
  }
  return plans
}

function createRequest(plan: GroupPlan, size: Size, context: PlacementContext) {
  const { gap, lineGap } = context.options
  const { direction, anchor } = plan
  const reach = Math.max(getReach(direction, size), plan.firstReach)
  const start = { x: anchor.x + direction.x * reach, y: anchor.y + direction.y * reach }
  // The middle of a group is counted in footprints: the names of a note fill the gap under its pill, so a note without names packs closer than one with.
  const cell = { width: size.width + gap, height: size.height + lineGap }
  const middle = getClusterMiddle(start, direction, { size: plan.members.length, cell })
  return { start, middle, direction, size, gaps: { gap, lineGap } }
}

function getPlacedNode(
  node: PlexNode,
  spot: Spot,
  context: PlacementContext,
  order: number,
): PlacedNode {
  return {
    ...node,
    x: spot.x,
    y: getPillCenterY(spot.y, context.room(node)),
    width: context.width(node),
    height: context.options.nodeSize.height,
    order,
    opacity: 1,
  }
}

function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const footprintOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const plans = createGroupPlans(nodesByRelation, focus, context, footprintOf)
  const index = new SpotIndex([
    { x: focus.x, y: focus.y, width: focus.width, height: focus.height },
  ])
  const groups = new Map<PlexRelation, Spot[]>()
  const placed = new Map<string, PlacedNode>()

  for (const node of takeTurns(nodesByRelation)) {
    const relation = node.relation as PlexRelation
    const plan = plans.get(relation)
    if (!plan) continue
    const group = groups.get(relation) ?? []
    groups.set(relation, group)

    const request = createRequest(plan, footprintOf(node), context)
    const spot = findFreeSpot(index, { ...request, group })
    index.add(spot)
    group.push(spot)
    const box = getPlacedNode(node, spot, context, plan.members.indexOf(node))
    placed.set(node.id, box)
    if (context.shouldStop?.(box)) break
  }

  return RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => placed.get(node.id) ?? []),
  )
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
