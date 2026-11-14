/**
 * Each side of the focus holds one block of a grid: the notes of the relations that grow to that side, row by row.
 * Every cell is as wide as the widest note and as tall as the tallest, so the blocks of all sides share one cell size.
 * The children block hangs under the focus and the parents block stands over it; a side block is centered on the
 * row of the focus, or grows up or down from it. The number of columns of each block is chosen so the blocks hold the most notes in the window and,
 * of those layouts, keep the notes nearest the focus.
 */
import { RELATION_NAMES, RELATIONS } from '../../relation'
import type { PlacedNode } from '../../node'
import type { Direction, PlexOptions } from '../options'
import {
  getFootprint,
  getPillCenterY,
  type NodesByRelation,
  type Placement,
  type PlacementContext,
} from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** One block: its notes in the order they fill its cells, and the side it grows to. */
interface Block {
  readonly side: Direction
  readonly members: readonly PlexNode[]
}

/** A block laid out: its column count and its bounds. */
interface BlockShape {
  readonly columns: number
  readonly rows: number
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

interface Cell {
  readonly width: number
  readonly height: number
}

interface Grid {
  readonly cell: Cell
  readonly gap: number
  readonly lineGap: number
  /** The distance from the center of the focus to the near edge of a side block, and to the near edge of a block over or under it. */
  readonly reach: { readonly x: number; readonly y: number }
  /** How far from the center a cell may reach and still be inside the window. */
  readonly window: { readonly x: number; readonly y: number }
}

/** The least room between two neighbors. */
const LEAST_GAP = 1e-6

const SIDES: readonly Direction[] = ['left', 'right', 'up', 'down']

const isAcross = (side: Direction): boolean => side === 'up' || side === 'down'

/** The notes of each side, the relations leaning up first, so the references stand over the jumps. */
function createBlocks(nodesByRelation: NodesByRelation, options: PlexOptions): Block[] {
  const leanRank = (relation: PlexRelation): number => {
    const { lean } = RELATIONS[relation]
    return lean === 'up' ? 0 : lean === null ? 1 : 2
  }
  return SIDES.flatMap((side) => {
    const relations = RELATION_NAMES.filter((relation) => options.direction[relation] === side)
    const members = [...relations]
      .sort((first, second) => leanRank(first) - leanRank(second))
      .flatMap((relation) => nodesByRelation[relation] ?? [])
    return members.length > 0 ? [{ side, members }] : []
  })
}

function getSize(columns: number, count: number, grid: Grid) {
  const rows = Math.ceil(count / columns)
  return {
    rows,
    width: columns * grid.cell.width + (columns - 1) * grid.gap,
    height: rows * grid.cell.height + (rows - 1) * grid.lineGap,
  }
}

/** Where a side block stands against the row of the focus: centered on it, or with its last or its first row on it. */
type Standing = 'center' | 'up' | 'down'

const STANDINGS: readonly Standing[] = ['center', 'up', 'down']

/** A side block, next to the focus. */
function shapeSideBlock(block: Block, columns: number, standing: Standing, grid: Grid): BlockShape {
  const { rows, width, height } = getSize(columns, block.members.length, grid)
  const near = grid.reach.x
  const [minX, maxX] = block.side === 'left' ? [-near - width, -near] : [near, near + width]
  const half = grid.cell.height / 2
  const minY = standing === 'up' ? half - height : standing === 'down' ? -half : -height / 2
  return { columns, rows, minX, maxX, minY, maxY: minY + height }
}

/** A block over or under the focus, centered on it, kept clear of the side blocks it would reach into. */
function shapeAcrossBlock(
  block: Block,
  columns: number,
  sideShapes: readonly BlockShape[],
  grid: Grid,
): BlockShape {
  const { rows, width, height } = getSize(columns, block.members.length, grid)
  const minX = -width / 2
  const maxX = width / 2
  const crossed = sideShapes.filter(
    (shape) => minX < shape.maxX + grid.gap && shape.minX - grid.gap < maxX,
  )
  if (block.side === 'down') {
    const top = Math.max(grid.reach.y, ...crossed.map((shape) => shape.maxY + grid.lineGap))
    return { columns, rows, minX, maxX, minY: top, maxY: top + height }
  }
  const bottom = Math.min(-grid.reach.y, ...crossed.map((shape) => shape.minY - grid.lineGap))
  return { columns, rows, minX, maxX, minY: bottom - height, maxY: bottom }
}

interface CellCenter {
  readonly x: number
  readonly y: number
}

/** The places of a row of `count` cells, from the middle out: the middle first, then left of it, then right of it. */
function listFromMiddle(count: number): number[] {
  return Array.from({ length: count }, (_, index) => index).sort(
    (first, second) =>
      Math.abs(2 * first - (count - 1)) - Math.abs(2 * second - (count - 1)) || first - second,
  )
}

/**
 * The centers of the cells a block fills, in the order of its notes, the row nearest the focus first.
 * Over and under the focus a row fills from its middle out and a partial last row is centered; on a side a row fills from the focus out.
 */
function listCells(block: Block, shape: BlockShape, grid: Grid): CellCenter[] {
  const { cell, gap, lineGap } = grid
  const count = block.members.length
  const stepX = cell.width + gap
  const stepY = cell.height + lineGap
  const centers: CellCenter[] = []
  for (let index = 0; index < count; index++) {
    const row = Math.floor(index / shape.columns)
    const column = index % shape.columns
    const inRow = Math.min(shape.columns, count - row * shape.columns)
    const y =
      block.side === 'up'
        ? shape.maxY - cell.height / 2 - row * stepY
        : shape.minY + cell.height / 2 + row * stepY
    if (isAcross(block.side)) {
      const rowWidth = inRow * cell.width + (inRow - 1) * gap
      const place = listFromMiddle(inRow)[column]!
      centers.push({ x: -rowWidth / 2 + cell.width / 2 + place * stepX, y })
    } else {
      const offset = cell.width / 2 + column * stepX
      centers.push({ x: block.side === 'left' ? shape.maxX - offset : shape.minX + offset, y })
    }
  }
  return centers
}

/** How many columns a block over or under the focus takes before it takes more only to keep its notes in the window. */
const ACROSS_COLUMNS = 5

/** What a layout is chosen by: the most notes in the window, then the fewest columns beyond `ACROSS_COLUMNS`, then the notes nearest the focus. */
interface Score {
  readonly inside: number
  readonly excess: number
  readonly distance: number
}

function scoreCells(cells: readonly CellCenter[], grid: Grid): Score {
  const { cell, window } = grid
  let inside = 0
  let distance = 0
  for (const { x, y } of cells) {
    if (Math.abs(x) + cell.width / 2 <= window.x && Math.abs(y) + cell.height / 2 <= window.y)
      inside++
    distance += Math.hypot(x / (cell.width + grid.gap), y / (cell.height + grid.lineGap))
  }
  return { inside, excess: 0, distance }
}

const isBetter = (score: Score, best: Score | undefined): boolean =>
  !best ||
  score.inside > best.inside ||
  (score.inside === best.inside &&
    (score.excess < best.excess ||
      (score.excess === best.excess && score.distance < best.distance - 1e-9)))

const add = (first: Score, second: Score): Score => ({
  inside: first.inside + second.inside,
  excess: first.excess + second.excess,
  distance: first.distance + second.distance,
})

/** How many columns a block may take: as many as reach the edge of the window, and no more than it has notes. */
function getMaxColumns(block: Block, grid: Grid): number {
  const { cell, gap, reach, window } = grid
  const room = isAcross(block.side) ? 2 * window.x : window.x - reach.x
  const fitting = Math.floor((room + gap) / (cell.width + gap))
  const square = Math.ceil(Math.sqrt(block.members.length)) * 2
  const limit = Number.isFinite(fitting) ? Math.max(1, fitting) : square
  return Math.max(1, Math.min(block.members.length, limit))
}

interface Plan {
  readonly block: Block
  readonly shape: BlockShape
  readonly cells: readonly CellCenter[]
}

/** A block laid out, kept by its side, columns and bounds: many choices of the side blocks leave a block over or under the focus where another choice did. */
type PlanCache = Map<string, { plan: Plan; score: Score }>

function planAcross(
  block: Block,
  sideShapes: readonly BlockShape[],
  grid: Grid,
  cache: PlanCache,
): { plan: Plan; score: Score } {
  let best: { plan: Plan; score: Score } | undefined
  for (let columns = 1; columns <= getMaxColumns(block, grid); columns++) {
    const shape = shapeAcrossBlock(block, columns, sideShapes, grid)
    const key = `${block.side}:${columns}:${shape.minY}`
    let planned = cache.get(key)
    if (!planned) {
      const cells = listCells(block, shape, grid)
      const score = {
        ...scoreCells(cells, grid),
        excess: Math.max(0, columns - ACROSS_COLUMNS),
      }
      planned = { plan: { block, shape, cells }, score }
      cache.set(key, planned)
    }
    if (isBetter(planned.score, best?.score)) best = planned
  }
  return best!
}

/** Every choice of columns for the side blocks, each with the best columns for the blocks over and under the focus. */
function planBlocks(blocks: readonly Block[], grid: Grid): Plan[] {
  const sides = blocks.filter((block) => !isAcross(block.side))
  const across = blocks.filter((block) => isAcross(block.side))
  let best: { plans: Plan[]; score: Score } | undefined
  const cache: PlanCache = new Map()
  const sidePlans = sides.map((block) => {
    const options: { plan: Plan; score: Score }[] = []
    for (let columns = 1; columns <= getMaxColumns(block, grid); columns++) {
      for (const standing of STANDINGS) {
        const shape = shapeSideBlock(block, columns, standing, grid)
        const cells = listCells(block, shape, grid)
        options.push({ plan: { block, shape, cells }, score: scoreCells(cells, grid) })
      }
    }
    return options
  })
  const choose = (index: number, chosen: readonly Plan[], chosenScore: Score) => {
    if (index === sides.length) {
      const shapes = chosen.map((plan) => plan.shape)
      let score = chosenScore
      const plans = [...chosen]
      for (const block of across) {
        const planned = planAcross(block, shapes, grid, cache)
        plans.push(planned.plan)
        score = add(score, planned.score)
      }
      if (isBetter(score, best?.score)) best = { plans, score }
      return
    }
    for (const option of sidePlans[index]!) {
      choose(index + 1, [...chosen, option.plan], add(chosenScore, option.score))
    }
  }
  choose(0, [], { inside: 0, excess: 0, distance: 0 })
  return best?.plans ?? []
}

function createGrid(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
  footprintOf: (node: PlexNode) => { width: number; height: number },
): Grid {
  const { gap, lineGap, focusGap, viewport, margin } = context.options
  const footprints = Object.values(nodesByRelation).flat().map(footprintOf)
  const cell = {
    width: Math.max(...footprints.map((footprint) => footprint.width)),
    height: Math.max(...footprints.map((footprint) => footprint.height)),
  }
  return {
    cell,
    // With no gap, neighbors keep a sliver apart so they do not overlap on float error.
    gap: Math.max(gap, LEAST_GAP),
    lineGap: Math.max(lineGap, LEAST_GAP),
    // A side block also keeps clear of one column over and under the focus, so a note there does not push it away.
    reach: {
      x: Math.max(focus.width / 2 + focusGap, cell.width / 2 + gap),
      y: focus.height / 2 + focusGap,
    },
    window: viewport
      ? { x: viewport.width / 2 - margin, y: viewport.height / 2 - margin }
      : { x: Infinity, y: Infinity },
  }
}

function placeGrid(
  nodesByRelation: NodesByRelation,
  focus: PlacedNode,
  context: PlacementContext,
): PlacedNode[] {
  const footprintOf = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const blocks = createBlocks(nodesByRelation, context.options)
  if (blocks.length === 0) return []
  const grid = createGrid(nodesByRelation, focus, context, footprintOf)
  const placed = new Map<string, PlacedNode>()
  const orders = new Map<PlexRelation, number>()
  for (const { block, cells } of planBlocks(blocks, grid)) {
    block.members.forEach((node, index) => {
      const cell = cells[index]!
      const footprint = footprintOf(node)
      // A footprint stands at the top of its cell, so the pills of one row line up.
      const footprintY = cell.y - grid.cell.height / 2 + footprint.height / 2
      const relation = node.relation as PlexRelation
      const order = orders.get(relation) ?? 0
      orders.set(relation, order + 1)
      placed.set(node.id, {
        ...node,
        x: cell.x,
        y: getPillCenterY(footprintY, context.room(node)),
        width: context.width(node),
        height: context.options.nodeSize.height,
        order,
        opacity: 1,
      })
    })
  }
  const ordered = RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => placed.get(node.id) ?? []),
  )
  const stopAt = context.shouldStop ? ordered.findIndex((node) => context.shouldStop!(node)) : -1
  return stopAt === -1 ? ordered : ordered.slice(0, stopAt + 1)
}

export const clusters: Placement = { name: 'clusters', place: placeGrid }
