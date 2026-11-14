/** Each relation grows as a compact cluster from its side of the focus, one note per relation in turn. */
import { RELATION_NAMES, RELATIONS, type Lean } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Direction, PlexOptions } from '../options'
import { findFreeSpot, SpotIndex, type Rect, type Spot, type Vector } from './freeSpot'
import { LineIndex } from './lines'
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

function getClusterMiddle(start: Vector, direction: Vector, request: MiddleRequest): Vector {
  // The middle of a group sits as far out from its start as half the patch is deep.
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

/** Two boxes closer than this still count as apart, so a spot built one gap from a group is not refused on float error. */
const TOLERANCE = 0.5

/** The boxes each group may not reach into: the bounds of every larger group, one gap wider on each side. */
type Zones = ReadonlyMap<PlexRelation, readonly Rect[]>

/**
 * The groups are laid out once, and again where a group ended up inside the bounds of a larger one: the smaller group is
 * then set where it keeps out of those bounds (cluster overlap removal, which moves the cluster with fewer notes).
 * Where no spot apart is free, the note takes the nearest free spot. Groups are kept apart only in a window, which bounds where a group may move to.
 */
function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  if (!context.isKeptApart || !context.fits)
    return placeGroups(nodesByRelation, focus, context).placed
  const first = placeGroups(nodesByRelation, focus, { ...context, shouldStop: undefined })
  const zones = getZones(nodesByRelation, first.groups, context.options)
  const isInsideZone = [...first.groups].some(([relation, spots]) =>
    spots.some((spot) => !isApart(spot, zones.get(relation))),
  )
  if (!isInsideZone) return first.placed
  return placeGroups(nodesByRelation, focus, context, zones).placed
}

function getBounds(spots: readonly Spot[]): Rect {
  return {
    minX: Math.min(...spots.map((spot) => spot.x - spot.width / 2)),
    maxX: Math.max(...spots.map((spot) => spot.x + spot.width / 2)),
    minY: Math.min(...spots.map((spot) => spot.y - spot.height / 2)),
    maxY: Math.max(...spots.map((spot) => spot.y + spot.height / 2)),
  }
}

function getZones(
  nodesByRelation: NodesByRelation,
  groups: ReadonlyMap<PlexRelation, readonly Spot[]>,
  { gap, lineGap }: PlexOptions,
): Zones {
  const countOf = (relation: PlexRelation) => nodesByRelation[relation]?.length ?? 0
  const zones = new Map<PlexRelation, Rect[]>()
  for (const relation of groups.keys()) {
    const larger = [...groups].filter(
      ([other, spots]) => countOf(other) > countOf(relation) && spots.length > 0,
    )
    zones.set(
      relation,
      larger.map(([, spots]) => {
        const bounds = getBounds(spots)
        return {
          minX: bounds.minX - gap,
          maxX: bounds.maxX + gap,
          minY: bounds.minY - lineGap,
          maxY: bounds.maxY + lineGap,
        }
      }),
    )
  }
  return zones
}

function isApart(spot: Spot, zones: readonly Rect[] | undefined): boolean {
  return (zones ?? []).every(
    (zone) =>
      spot.x + spot.width / 2 <= zone.minX + TOLERANCE ||
      spot.x - spot.width / 2 >= zone.maxX - TOLERANCE ||
      spot.y + spot.height / 2 <= zone.minY + TOLERANCE ||
      spot.y - spot.height / 2 >= zone.maxY - TOLERANCE,
  )
}

function placeGroups(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
  zones?: Zones,
): { placed: PlacedNode[]; groups: ReadonlyMap<PlexRelation, readonly Spot[]> } {
  const footprintOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const plans = createGroupPlans(nodesByRelation, focus, context, footprintOf)
  const focusSpot: Spot = { x: focus.x, y: focus.y, width: focus.width, height: focus.height }
  const index = new SpotIndex([focusSpot])
  const lines = new LineIndex(context.edges ?? [], focus.id, index)
  lines.place(focus.id, { x: focus.x, y: focus.y }, focusSpot)
  const groups = new Map<PlexRelation, Spot[]>()
  const placed = new Map<string, PlacedNode>()

  for (const node of takeTurns(nodesByRelation)) {
    const relation = node.relation as PlexRelation
    const plan = plans.get(relation)
    if (!plan) continue
    const group = groups.get(relation) ?? []
    groups.set(relation, group)

    const request = createRequest(plan, footprintOf(node), context)
    const room = context.room(node)
    const getCenter = (at: Spot): Vector => ({ x: at.x, y: getPillCenterY(at.y, room) })
    const avoided = zones?.get(relation)
    const spot = findFreeSpot(index, {
      ...request,
      group,
      fits: context.fits,
      isLineClear: (at) => lines.isClear(node.id, getCenter(at), at),
      ...(avoided?.length ? { isApart: (at: Spot) => isApart(at, avoided) } : {}),
    })
    index.add(spot)
    lines.place(node.id, getCenter(spot), spot)
    const box = getPlacedNode(node, spot, context, group.length)
    group.push(spot)
    placed.set(node.id, box)
    if (context.shouldStop?.(box)) break
  }

  return {
    placed: RELATION_NAMES.flatMap((relation) =>
      (nodesByRelation[relation] ?? []).flatMap((node) => placed.get(node.id) ?? []),
    ),
    groups,
  }
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
