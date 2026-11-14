/** Each relation fills a sector around the focus ring by ring outward; the sector angles follow the note counts. */
import { RELATION_NAMES } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Size } from '../options'
import { LineIndex } from './lines'
import {
  getFootprint,
  getPillCenterY,
  type NodesByRelation,
  type Placement,
  type PlacementContext,
} from './placement'
import { getSectors, layRing, type Sector } from './sectors'
import { SpotIndex, type Spot, type Vector } from './spotIndex'
import type { PlexNode, PlexRelation } from '../../../../types'

/** The rings of a sector stop here when there is no window to end them. */
const MAX_RINGS = 400

/** What one placement shares across its sectors. */
interface Run {
  readonly focus: PlacedNode
  readonly context: PlacementContext
  readonly index: SpotIndex
  readonly lines: LineIndex
  readonly placed: Map<string, PlacedNode>
}

const getReach = (viewport: Size): number => Math.hypot(viewport.width, viewport.height)

function isInWindow(spot: Spot, context: PlacementContext): boolean {
  const { viewport, margin } = context.options
  if (context.fits && !context.fits(spot)) return false
  return (
    !viewport ||
    (Math.abs(spot.x) + spot.width / 2 <= viewport.width / 2 - margin &&
      Math.abs(spot.y) + spot.height / 2 <= viewport.height / 2 - margin)
  )
}

function getFootprintOf(node: PlexNode, context: PlacementContext): Size {
  return getFootprint(
    { width: context.width(node), height: context.options.nodeSize.height },
    context.room(node),
  )
}

/** Where a note goes and its place in the order of its group. */
interface Seat {
  readonly spot: Spot
  readonly order: number
}

function place(run: Run, node: PlexNode, { spot, order }: Seat): boolean {
  const { context } = run
  const room = context.room(node)
  const y = getPillCenterY(spot.y, room)
  const box: PlacedNode = {
    ...node,
    x: spot.x,
    y,
    width: context.width(node),
    height: context.options.nodeSize.height,
    order,
    opacity: 1,
  }
  run.index.add(spot)
  run.lines.place(node.id, { x: spot.x, y }, spot)
  run.placed.set(node.id, box)
  return context.shouldStop?.(box) ?? false
}

function getLeftOutSpot(sector: Sector, cell: Size, context: PlacementContext): Spot {
  const middle = (((sector.from + sector.to) / 2) * Math.PI) / 180
  const { viewport } = context.options
  const radius = (viewport ? getReach(viewport) : 0) + cell.width
  return { x: radius * Math.cos(middle), y: radius * Math.sin(middle), ...cell }
}

/** Fills the sector ring by ring; returns whether a caller asked to stop. */
function placeSector(run: Run, sector: Sector, members: readonly PlexNode[]): boolean {
  const { context } = run
  const { gap, lineGap, focusGap, viewport } = context.options
  const sizes = members.map((node) => getFootprintOf(node, context))
  const height = Math.max(...sizes.map((size) => size.height))
  const first = run.focus.height / 2 + focusGap + height / 2
  const step = height + lineGap
  const maxRadius = viewport ? getReach(viewport) / 2 : Number.POSITIVE_INFINITY
  const placedBefore = run.placed.size
  let next = 0
  let isStopped = false
  for (let ring = 0; next < members.length && ring < MAX_RINGS; ring++) {
    const radius = first + ring * step
    if (radius > maxRadius) break
    const rest = members.slice(next)
    layRing(run.index, {
      sector,
      radius,
      isOdd: ring % 2 === 1,
      gaps: { gap, lineGap },
      isInside: (spot) => isInWindow(spot, context),
      isLineClear: (position, spot) => isLineClear(run, rest[position]!, spot),
      sizes: sizes.slice(next),
      commit: (position, spot) => {
        isStopped = place(run, rest[position]!, { spot, order: next + position }) || isStopped
      },
    })
    next = run.placed.size - placedBefore
    if (isStopped) return true
  }
  for (; next < members.length; next++) {
    const node = members[next]!
    const spot = getLeftOutSpot(sector, getFootprintOf(node, context), context)
    if (place(run, node, { spot, order: next })) return true
  }
  return false
}

function isLineClear(run: Run, node: PlexNode, spot: Spot): boolean {
  const center = { x: spot.x, y: getPillCenterY(spot.y, run.context.room(node)) }
  return run.lines.isClear(node.id, center, spot)
}

function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const { gap, lineGap, focusGap } = context.options
  const counts: Partial<Record<PlexRelation, number>> = {}
  for (const relation of RELATION_NAMES) counts[relation] = nodesByRelation[relation]?.length
  // The focus keeps `focusGap` around it, where other boxes keep `gap`.
  const focusSpot: Spot = {
    x: focus.x,
    y: focus.y,
    width: focus.width + 2 * Math.max(0, focusGap - gap),
    height: focus.height + 2 * Math.max(0, focusGap - lineGap),
  }
  const index = new SpotIndex([focusSpot])
  const lines = new LineIndex(context.edges ?? [], focus.id, index)
  lines.place(focus.id, { x: focus.x, y: focus.y } satisfies Vector, focusSpot)
  const run: Run = { focus, context, index, lines, placed: new Map() }
  for (const sector of getSectors(counts)) {
    if (placeSector(run, sector, nodesByRelation[sector.relation] ?? [])) break
  }
  return RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => run.placed.get(node.id) ?? []),
  )
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
