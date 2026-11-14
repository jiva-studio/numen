/**
 * Each side of the focus holds one block of a grid: the notes of the relations that grow to that side, row by row.
 * A block's cells are as wide as its widest note and as tall as its tallest, so a block of short titles stays compact.
 * The children block hangs right under the focus and the parents block stands right over it; a side block is
 * centered on the row of the focus, or grows up or down from it, and keeps out of the columns of the blocks over and
 * under the focus. The number of columns of each block is chosen so the blocks hold the most notes in the window and,
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

interface Size {
  readonly width: number
  readonly height: number
}

/** One block: its notes in the order they fill its cells, the side it grows to, and the size of its cells. */
interface Block {
  readonly side: Direction
  readonly members: readonly PlexNode[]
  readonly cell: Size
}

/** A block laid out: its column count and its bounds. */
interface BlockShape {
  readonly columns: number
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

interface Grid {
  readonly gap: number
  readonly lineGap: number
  /** The distance from the center of the focus to the near edge of a side block, and to the near edge of a block over or under it. */
  readonly reach: { readonly x: number; readonly y: number }
  /** How far from the center a cell may reach and still be inside the window. */
  readonly window: { readonly x: number; readonly y: number }
}

/** Blocks that touch within this much do not count as reaching each other, so float error does not push a block away. */
const TOLERANCE = 1e-6

/** The least room between two neighbors. */
const LEAST_GAP = 1e-6

/** How many columns a block over or under the focus takes before it takes more only to keep its notes in the window. */
const ACROSS_COLUMNS = 5

const SIDES: readonly Direction[] = ['left', 'right', 'up', 'down']

const isAcross = (side: Direction): boolean => side === 'up' || side === 'down'

/** The notes of each side, the relations leaning up first, so the references stand over the jumps. */
function createBlocks(
  nodesByRelation: NodesByRelation,
  options: PlexOptions,
  footprintOf: (node: PlexNode) => Size,
): Block[] {
  const leanRank = (relation: PlexRelation): number => {
    const { lean } = RELATIONS[relation]
    return lean === 'up' ? 0 : lean === null ? 1 : 2
  }
  return SIDES.flatMap((side) => {
    const relations = RELATION_NAMES.filter((relation) => options.direction[relation] === side)
    const members = [...relations]
      .sort((first, second) => leanRank(first) - leanRank(second))
      .flatMap((relation) => nodesByRelation[relation] ?? [])
    if (members.length === 0) return []
    const footprints = members.map(footprintOf)
    const cell = {
      width: Math.max(...footprints.map((footprint) => footprint.width)),
      height: Math.max(...footprints.map((footprint) => footprint.height)),
    }
    return [{ side, members, cell }]
  })
}

function getSize(block: Block, columns: number, grid: Grid): Size {
  const rows = Math.ceil(block.members.length / columns)
  return {
    width: columns * block.cell.width + (columns - 1) * grid.gap,
    height: rows * block.cell.height + (rows - 1) * grid.lineGap,
  }
}

/** A block right over or under the focus, centered on it. */
function shapeAcrossBlock(block: Block, columns: number, grid: Grid): BlockShape {
  const { width, height } = getSize(block, columns, grid)
  const [minY, maxY] =
    block.side === 'down'
      ? [grid.reach.y, grid.reach.y + height]
      : [-grid.reach.y - height, -grid.reach.y]
  return { columns, minX: -width / 2, maxX: width / 2, minY, maxY }
}

/** Where a side block stands against the focus: centered on its row, or ending just over the room of the children, or starting just under the room of the parents. */
type Standing = 'center' | 'up' | 'down'

const STANDINGS: readonly Standing[] = ['center', 'up', 'down']

/** A side block next to the focus, past the columns of every block over or under the focus that its rows reach. */
function shapeSideBlock(
  block: Block,
  { columns, standing }: { readonly columns: number; readonly standing: Standing },
  acrossShapes: readonly BlockShape[],
  grid: Grid,
): BlockShape {
  const { width, height } = getSize(block, columns, grid)
  const edge = grid.reach.y - grid.lineGap
  const [minY, maxY] =
    standing === 'up'
      ? [edge - height, edge]
      : standing === 'down'
        ? [-edge, -edge + height]
        : [-height / 2, height / 2]
  const reached = acrossShapes.filter(
    (shape) =>
      minY < shape.maxY + grid.lineGap - TOLERANCE && shape.minY - grid.lineGap + TOLERANCE < maxY,
  )
  const near = Math.max(grid.reach.x, ...reached.map((shape) => shape.maxX + grid.gap))
  const [minX, maxX] = block.side === 'left' ? [-near - width, -near] : [near, near + width]
  return { columns, minX, maxX, minY, maxY }
}

/** The places of a row of `count` cells, from the middle out: the middle first, then left of it, then right of it. */
function listFromMiddle(count: number): number[] {
  return Array.from({ length: count }, (_, index) => index).sort(
    (first, second) =>
      Math.abs(2 * first - (count - 1)) - Math.abs(2 * second - (count - 1)) || first - second,
  )
}

interface CellCenter {
  readonly x: number
  readonly y: number
}

/**
 * The centers of the cells a block fills, in the order of its notes, the row nearest the focus first.
 * Over and under the focus a row fills from its middle out and a partial last row is centered; on a side a row fills from the focus out.
 */
function listCells(block: Block, shape: BlockShape, grid: Grid): CellCenter[] {
  const { cell } = block
  const count = block.members.length
  const stepX = cell.width + grid.gap
  const stepY = cell.height + grid.lineGap
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
      const rowWidth = inRow * cell.width + (inRow - 1) * grid.gap
      const place = listFromMiddle(inRow)[column]!
      centers.push({ x: -rowWidth / 2 + cell.width / 2 + place * stepX, y })
    } else {
      const offset = cell.width / 2 + column * stepX
      centers.push({ x: block.side === 'left' ? shape.maxX - offset : shape.minX + offset, y })
    }
  }
  return centers
}

/** What a layout is chosen by: the most notes in the window, then the fewest columns beyond `ACROSS_COLUMNS`, then the notes nearest the focus. */
interface Score {
  readonly inside: number
  readonly excess: number
  readonly distance: number
}

const NO_SCORE: Score = { inside: 0, excess: 0, distance: 0 }

function scoreCells(block: Block, cells: readonly CellCenter[], grid: Grid): Score {
  const { cell } = block
  const { window } = grid
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
  const { gap, reach, window } = grid
  const room = isAcross(block.side) ? 2 * window.x : window.x - reach.x
  const fitting = Math.floor((room + gap) / (block.cell.width + gap))
  const square = Math.ceil(Math.sqrt(block.members.length)) * 2
  const limit = Number.isFinite(fitting) ? Math.max(1, fitting) : square
  return Math.max(1, Math.min(block.members.length, limit))
}

interface Plan {
  readonly block: Block
  readonly cells: readonly CellCenter[]
}

interface Scored {
  readonly plan: Plan
  readonly shape: BlockShape
  readonly score: Score
}

function listColumnCounts(block: Block, grid: Grid): number[] {
  return Array.from({ length: getMaxColumns(block, grid) }, (_, index) => index + 1)
}

function planAcross(block: Block, columns: number, grid: Grid): Scored {
  const shape = shapeAcrossBlock(block, columns, grid)
  const cells = listCells(block, shape, grid)
  const score = {
    ...scoreCells(block, cells, grid),
    excess: Math.max(0, columns - ACROSS_COLUMNS),
  }
  return { plan: { block, cells }, shape, score }
}

/** The best columns and standing of a side block, given the blocks over and under the focus. */
function planSide(block: Block, acrossShapes: readonly BlockShape[], grid: Grid): Scored {
  let best: Scored | undefined
  for (const columns of listColumnCounts(block, grid)) {
    for (const standing of STANDINGS) {
      const shape = shapeSideBlock(block, { columns, standing }, acrossShapes, grid)
      const cells = listCells(block, shape, grid)
      const score = scoreCells(block, cells, grid)
      if (isBetter(score, best?.score)) best = { plan: { block, cells }, shape, score }
    }
  }
  return best!
}

/** Every choice of columns for the blocks over and under the focus, each with the best side blocks it leaves room for. */
function planBlocks(blocks: readonly Block[], grid: Grid): Plan[] {
  const across = blocks.filter((block) => isAcross(block.side))
  const sides = blocks.filter((block) => !isAcross(block.side))
  const acrossChoices = across.map((block) =>
    listColumnCounts(block, grid).map((columns) => planAcross(block, columns, grid)),
  )
  let best: { plans: Plan[]; score: Score } | undefined
  const choose = (index: number, chosen: readonly Scored[]) => {
    if (index < across.length) {
      for (const option of acrossChoices[index]!) choose(index + 1, [...chosen, option])
      return
    }
    const shapes = chosen.map((one) => one.shape)
    const all = [...chosen, ...sides.map((block) => planSide(block, shapes, grid))]
    const score = all.reduce((total, one) => add(total, one.score), NO_SCORE)
    if (isBetter(score, best?.score)) best = { plans: all.map((one) => one.plan), score }
  }
  choose(0, [])
  return best?.plans ?? []
}

function createGrid(focus: PlacedNode, options: PlexOptions): Grid {
  const { gap, lineGap, focusGap, viewport, margin } = options
  return {
    // With no gap, neighbors keep a sliver apart so they do not overlap on float error.
    gap: Math.max(gap, LEAST_GAP),
    lineGap: Math.max(lineGap, LEAST_GAP),
    reach: { x: focus.width / 2 + focusGap, y: focus.height / 2 + focusGap },
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
  const blocks = createBlocks(nodesByRelation, context.options, footprintOf)
  if (blocks.length === 0) return []
  const grid = createGrid(focus, context.options)
  const placed = new Map<string, PlacedNode>()
  const orders = new Map<PlexRelation, number>()
  for (const { block, cells } of planBlocks(blocks, grid)) {
    block.members.forEach((node, index) => {
      const cell = cells[index]!
      const footprint = footprintOf(node)
      // A footprint stands at the top of its cell, so the pills of one row line up.
      const footprintY = cell.y - block.cell.height / 2 + footprint.height / 2
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
