/** One even grid around the focus, and each relation takes the cells of its sector nearest the focus; the sector angles follow the note counts. */
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
import { getSectors, listSectorCells, type Cell, type Sector } from './sectors'
import { isClear, SpotIndex, type Gaps, type Spot, type Vector } from './spotIndex'
import type { PlexNode, PlexRelation } from '../../../../types'

/** Two lines from the focus closer than this (the sine of the angle between them) to one direction read as one line. */
const ALONG_SINE = 0.12

/**
 * How much farther than the nearest free cell a note goes to keep its lines apart; beyond it the block stays compact. A
 * note joined to a placed note other than the focus goes farther: its line runs between two notes of the plex.
 */
const LINE_REACH = 0.5
const LINE_REACH_BEYOND_HUB = 2

/** What one placement shares across its sectors. */
interface Run {
  readonly focusHeight: number
  readonly focusSpot: Spot
  readonly context: PlacementContext
  readonly gaps: Gaps
  /** The distance between two columns and between two rows of the grid, the same for every group. */
  readonly step: Size
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

/** Whether the line from the focus to the note would run along the line to a note placed before, on the same side. */
function isAlong(run: Run, center: Vector): boolean {
  const length = Math.hypot(center.x, center.y)
  for (const other of run.placed.values()) {
    const cross = center.x * other.y - center.y * other.x
    const dot = center.x * other.x + center.y * other.y
    if (dot > 0 && Math.abs(cross) < ALONG_SINE * length * Math.hypot(other.x, other.y)) return true
  }
  return false
}

function getPillCenter(run: Run, node: PlexNode, spot: Spot): Vector {
  return { x: spot.x, y: getPillCenterY(spot.y, run.context.room(node)) }
}

/** The cells of a sector, nearest first, and the ones already taken. */
interface SectorCells {
  readonly cells: readonly Cell[]
  readonly used: Set<number>
}

/** A cell a note can take, and the note's box in it. */
interface Candidate {
  readonly position: number
  readonly spot: Spot
}

/** The free cells for a note of this size, from the nearest free one to `reach` columns farther. */
function listFreeCells(run: Run, size: Size, sector: SectorCells, reach: number): Candidate[] {
  const free: Candidate[] = []
  for (const [position, cell] of sector.cells.entries()) {
    if (sector.used.has(position)) continue
    const spot = { x: cell.x, y: cell.y, ...size }
    if (!isInWindow(spot, run.context) || !run.index.isFree(spot, run.gaps)) continue
    const nearest = free[0] && sector.cells[free[0].position]!
    if (nearest && cell.distance > nearest.distance + reach * run.step.width) break
    free.push({ position, spot })
  }
  return free
}

/**
 * The cell the note takes, among the cells not much farther than the nearest free one: the first off the lines from the
 * focus whose lines keep clear of the other lines and notes, then the first whose lines at least run through no note, then
 * the first off the lines from the focus. Failing those, the nearest free cell.
 */
function findCell(run: Run, node: PlexNode, sector: SectorCells): Candidate | undefined {
  const reach = run.lines.hasLineBeyondHub(node.id) ? LINE_REACH_BEYOND_HUB : LINE_REACH
  const free = listFreeCells(run, getFootprintOf(node, run.context), sector, reach)
  const offLine = free.filter(({ spot }) => !isAlong(run, getPillCenter(run, node, spot)))
  const isClearAt =
    (clearance?: number) =>
    ({ spot }: Candidate): boolean =>
      run.lines.isClear(node.id, getPillCenter(run, node, spot), spot, clearance)
  return offLine.find(isClearAt()) ?? offLine.find(isClearAt(0)) ?? offLine[0] ?? free[0]
}

/** Fills the cells of the sector, nearest first; returns whether a caller asked to stop. */
function placeSector(run: Run, sector: Sector, members: readonly PlexNode[]): boolean {
  const { context, step, gaps } = run
  const { viewport, margin, focusGap } = context.options
  const footprint = { width: step.width - gaps.gap, height: step.height - gaps.lineGap }
  const cells = listSectorCells({
    sector,
    step,
    firstRow: run.focusHeight / 2 + focusGap + footprint.height / 2,
    count: members.length,
    bounds: viewport && {
      width: viewport.width / 2 - margin,
      height: viewport.height / 2 - margin,
    },
    isUsable: (cell) => isClear({ ...cell, ...footprint }, run.focusSpot, gaps),
  })
  const taken: SectorCells = { cells, used: new Set() }
  for (const [order, node] of members.entries()) {
    const found = findCell(run, node, taken)
    if (found) taken.used.add(found.position)
    const spot = found?.spot ?? getLeftOutSpot(sector, getFootprintOf(node, context), context)
    if (place(run, node, { spot, order })) return true
  }
  return false
}

/** The widest and the tallest footprint of all notes, and the gaps, make one step of the grid for every group. */
function getStep(nodesByRelation: NodesByRelation, context: PlacementContext): Size {
  const { gap, lineGap } = context.options
  let width = 0
  let height = 0
  for (const nodes of Object.values(nodesByRelation)) {
    for (const node of nodes ?? []) {
      const size = getFootprintOf(node, context)
      width = Math.max(width, size.width)
      height = Math.max(height, size.height)
    }
  }
  return { width: width + gap, height: height + lineGap }
}

function placeClusters(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const { gap, lineGap, focusGap } = context.options
  const counts: Partial<Record<PlexRelation, number>> = {}
  for (const relation of RELATION_NAMES) counts[relation] = nodesByRelation[relation]?.length ?? 0
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
  const run: Run = {
    focusHeight: focus.height,
    focusSpot,
    context,
    gaps: { gap, lineGap },
    step: getStep(nodesByRelation, context),
    index,
    lines,
    placed: new Map(),
  }
  for (const sector of getSectors(counts)) {
    if (placeSector(run, sector, nodesByRelation[sector.relation] ?? [])) break
  }
  return RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => run.placed.get(node.id) ?? []),
  )
}

export const clusters: Placement = { name: 'clusters', place: placeClusters }
