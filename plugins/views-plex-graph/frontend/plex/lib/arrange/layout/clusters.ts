/** Each relation grows as a compact cluster from its side of the focus, one note per relation in turn. */
import { RELATION_NAMES, RELATIONS, type Lean } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Direction, PlexOptions } from '../options'
import {
  findFreeSpot,
  findSpotWithin,
  SpotIndex,
  type Spot,
  type SpotRequest,
  type Vector,
} from './freeSpot'
import {
  findBestAttempt,
  getStartColumns,
  isInWindow,
  measureMiddleBand,
  type Attempt,
} from './bands'
import { LineIndex } from './lines'
import { createBorders, createTerritory, GROUP_GAP_FACTOR, type Borders } from './territory'
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

/** A lane opens by this much across its axis for each unit it runs along it; a row opens wider by how many times a cell is wider than tall. */
const LANE_SLOPE = 1

/** A center may lie this far short of the edge of its lane, so a box built exactly on the edge is inside. */
const LANE_TOLERANCE = 0.5

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
  readonly relation: PlexRelation
  readonly members: readonly PlexNode[]
  readonly side: Vector
  readonly direction: Vector
  readonly anchor: Vector
  readonly firstReach: number
  readonly isSide: boolean
  readonly hasLane: boolean
  readonly isFenced: boolean
  readonly isInTerritory: (spot: Spot) => boolean
}

const getReach = (direction: Vector, box: { width: number; height: number }): number =>
  Math.abs(direction.x) * (box.width / 2) + Math.abs(direction.y) * (box.height / 2)

interface PlanContext {
  readonly focus: PlacedNode
  readonly context: PlacementContext
  readonly footprintOf: (node: PlexNode) => Size
  readonly borders: Borders
}

function createGroupPlans(
  nodesByRelation: NodesByRelation,
  { focus, context, footprintOf, borders }: PlanContext,
): ReadonlyMap<PlexRelation, GroupPlan> {
  const plans = new Map<PlexRelation, GroupPlan>()
  for (const relation of RELATION_NAMES) {
    const members = nodesByRelation[relation] ?? []
    const first = members[0]
    if (!first) continue
    const side = context.options.direction[relation]
    // A group leans only where another group shares its side, so the two part; alone it grows straight out.
    const isShared = RELATION_NAMES.some(
      (other) =>
        other !== relation &&
        (nodesByRelation[other]?.length ?? 0) > 0 &&
        context.options.direction[other] === side,
    )
    const direction = getClusterDirection(side, isShared ? RELATIONS[relation].lean : null)
    const isSide = SIDES[side].x !== 0
    const anchor = getAnchor(direction, focus, context.options)
    plans.set(relation, {
      relation,
      members,
      side: SIDES[side],
      direction,
      anchor: isSide ? { x: SIDES[side].x * borders.sideStart, y: anchor.y } : anchor,
      firstReach: getReach(direction, footprintOf(first)),
      isSide,
      hasLane: !isSide || borders.isOpen,
      isFenced: !borders.isOpen,
      isInTerritory: createTerritory(SIDES[side], focus, borders),
    })
  }
  return plans
}

function createLane(
  plan: GroupPlan,
  focus: PlacedNode,
  cell: Size,
): ((point: Vector) => boolean) | undefined {
  const { side, anchor } = plan
  if (!plan.hasLane) return undefined
  const isRow = side.x === 0
  const slope = isRow ? (LANE_SLOPE * cell.width) / cell.height : LANE_SLOPE
  const base = isRow ? focus.width / 2 + 2 * cell.width : focus.height / 2 + 2 * cell.height
  return (point) => {
    const along = side.x * (point.x - anchor.x) + side.y * (point.y - anchor.y)
    const across = Math.abs(isRow ? point.x - focus.x : point.y - focus.y)
    return along >= -LANE_TOLERANCE && across <= base + slope * along
  }
}

function createRequest(plan: GroupPlan, focus: PlacedNode, size: Size, context: PlacementContext) {
  const { gap, lineGap } = context.options
  const { direction, anchor } = plan
  const reach = Math.max(getReach(direction, size), plan.firstReach)
  // A box of a strip begins with its edge on the border of the strip.
  const start = {
    x: plan.isSide ? anchor.x + plan.side.x * (size.width / 2) : anchor.x + direction.x * reach,
    y: anchor.y + direction.y * reach,
  }
  // The middle of a group is counted in footprints: the names of a note fill the gap under its pill, so a note without names packs closer than one with.
  const cell = { width: size.width + gap, height: size.height + lineGap }
  const middle = getClusterMiddle(start, direction, { size: plan.members.length, cell })
  return {
    start,
    middle,
    direction,
    size,
    gaps: { gap, lineGap },
    isInLane: createLane(plan, focus, cell),
    isInTerritory: plan.isInTerritory,
  }
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

function findNoteSpot(index: SpotIndex, request: SpotRequest, plan: GroupPlan): Spot | undefined {
  const within = findSpotWithin(index, request)
  if (plan.isFenced) return within
  if (within && request.isLineClear?.(within) !== false) return within
  const loose = findFreeSpot(index, { ...request, isInTerritory: undefined })
  return !within || request.isLineClear?.(loose) !== false ? loose : within
}

function getLeftOutSpot(plan: GroupPlan, size: Size, context: PlacementContext): Spot {
  const { viewport } = context.options
  const reach = viewport ? Math.max(viewport.width, viewport.height) : 0
  return {
    x: plan.side.x * (reach + size.width),
    y: plan.side.y * (reach + size.height),
    ...size,
  }
}

/** What one placement shares across its notes: the plans, the boxes and lines placed so far, and the tally of how the notes fared. */
interface Run {
  readonly focus: PlacedNode
  readonly context: PlacementContext
  readonly plans: ReadonlyMap<PlexRelation, GroupPlan>
  readonly footprintOf: (node: PlexNode) => Size
  readonly index: SpotIndex
  readonly lines: LineIndex
  readonly groups: Map<PlexRelation, Spot[]>
  readonly failed: Set<PlexRelation>
  readonly totals: { failures: number; crossings: number; outside: number; centralReach: number }
}

function tally(run: Run, plan: GroupPlan, spot: Spot, outcome: 'placed' | 'left-out' | 'crossing') {
  const { totals, failed } = run
  if (outcome !== 'placed') failed.add(plan.relation)
  if (outcome === 'left-out') totals.failures++
  if (outcome === 'crossing') totals.crossings++
  if (!isInWindow(spot, run.context.options)) totals.outside++
  if (outcome !== 'left-out' && !plan.isSide) {
    totals.centralReach = Math.max(totals.centralReach, Math.abs(spot.x) + spot.width / 2)
  }
}

function placeNote(run: Run, node: PlexNode): PlacedNode | undefined {
  const { focus, context, index, lines } = run
  const plan = run.plans.get(node.relation as PlexRelation)
  if (!plan) return undefined
  const group = run.groups.get(plan.relation) ?? []
  run.groups.set(plan.relation, group)
  const room = context.room(node)
  const getCenter = (at: Spot): Vector => ({ x: at.x, y: getPillCenterY(at.y, room) })
  const size = run.footprintOf(node)
  const isLineClear = (at: Spot) => lines.isClear(node.id, getCenter(at), at)
  // A fenced note stays in its territory: with no free spot there it is set beyond the window, and the window cuts it.
  const within = findNoteSpot(
    index,
    { ...createRequest(plan, focus, size, context), group, fits: context.fits, isLineClear },
    plan,
  )
  const spot = within ?? getLeftOutSpot(plan, size, context)
  const isCrossing = within !== undefined && plan.isFenced && !isLineClear(within)
  tally(run, plan, spot, !within ? 'left-out' : isCrossing ? 'crossing' : 'placed')
  const box = getPlacedNode(node, spot, context, group.length)
  if (within) {
    index.add(spot)
    lines.place(node.id, getCenter(spot), spot)
    group.push(spot)
  }
  return box
}

function placeInTerritories(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
  borders: Borders,
): Attempt {
  const footprintOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const focusSpot: Spot = { x: focus.x, y: focus.y, width: focus.width, height: focus.height }
  const index = new SpotIndex([focusSpot])
  const lines = new LineIndex(context.edges ?? [], focus.id, index)
  lines.place(focus.id, { x: focus.x, y: focus.y }, focusSpot)
  const run: Run = {
    focus,
    context,
    footprintOf,
    index,
    lines,
    plans: createGroupPlans(nodesByRelation, { focus, context, footprintOf, borders }),
    groups: new Map(),
    failed: new Set(),
    totals: { failures: 0, crossings: 0, outside: 0, centralReach: 0 },
  }
  const placed = new Map<string, PlacedNode>()
  for (const node of takeTurns(nodesByRelation)) {
    const box = placeNote(run, node)
    if (!box) continue
    placed.set(node.id, box)
    if (context.shouldStop?.(box)) break
  }
  return { placed, failed: run.failed, ...run.totals }
}

function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const { gap, focusGap } = context.options
  const band = measureMiddleBand(nodesByRelation, context)
  const bordersFor = (columns: number) => ({
    columns,
    cellWidth: band.cellWidth,
    hasCentral: band.hasCentral,
    hasSide: band.hasSide,
    groupGap: GROUP_GAP_FACTOR * gap,
    focusReach: focus.width / 2 + focusGap,
  })
  const placeAt = (borders: Borders) => placeInTerritories(nodesByRelation, focus, context, borders)

  const best = findBestAttempt(
    (columns) => placeAt(createBorders(bordersFor(columns))),
    band,
    context,
    getStartColumns(band, context, focus),
  )
  return RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => best.placed.get(node.id) ?? []),
  )
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
