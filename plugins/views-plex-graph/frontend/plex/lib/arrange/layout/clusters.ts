/**
 * The `clusters` placement. Each side of the focus holds one block of a grid: the notes of the relations that grow to
 * that side, row by row. A side that holds relations leaning up and relations that do not holds two blocks instead,
 * which share the cell size and the near column: the upper one gathers up from the row of the focus, its first row on
 * that row, and the lower one gathers down from the row under it, so the references stand over the jumps.
 *
 * A block's cells are as wide as its widest footprint and as tall as its tallest, and a footprint is aligned to the top of its
 * cell, so the pills of one row line up. The children block hangs under the focus and the parents block is placed over it,
 * both centered on the focus; their rows fill from the middle out and a partial last row is centered. A side block is
 * centered on the row of the focus, ends one line gap over the children block, or starts one line gap under the parents
 * block, unless the side holds two blocks; it is placed past the columns of every block over or under the focus that its
 * rows reach, and its rows fill from the focus out.
 *
 * The column counts and the side block positions are chosen together: the most notes inside the window first, then the
 * fewest columns a block over or under the focus takes beyond `ACROSS_COLUMNS`, then the least distance of the notes
 * from the focus, counted in cells. A block takes at most as many columns as reach the edge of the window; with no
 * window, twice the square root of its note count, rounded up.
 */
import { RELATION_NAMES, RELATIONS, type Lean } from '../../relation'
import type { PlacedNode, Position } from '../../node'
import type { Direction, PlexOptions, Size } from '../options'
import {
  getFootprint,
  getPillCenterY,
  type NodesByRelation,
  type Placement,
  type PlacementContext,
} from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** The notes of one side, or of its upper or lower half, in the order they fill the cells, and the size of every cell. */
interface Block {
  readonly side: Direction
  readonly half: Lean | null
  readonly members: readonly PlexNode[]
  readonly cell: Size
}

interface BlockShape {
  readonly columns: number
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

/**
 * The gaps, the distance from the center of the focus to the near edge of a side block (`reach.x`) and of a block over
 * or under it (`reach.y`), and how far from the center a cell may reach and stay inside the window.
 */
interface Grid {
  readonly gap: number
  readonly lineGap: number
  readonly reach: { readonly x: number; readonly y: number }
  readonly window: { readonly x: number; readonly y: number }
}

/** Blocks that touch within this distance do not count as reaching each other. */
const TOLERANCE = 1e-6

/** The least gap between two neighbors, so neighbors with no gap do not overlap on float error. */
const LEAST_GAP = 1e-6

/** The column count of a block over or under the focus past which it takes more columns only to show more notes. */
const ACROSS_COLUMNS = 5

const SIDES: readonly Direction[] = ['left', 'right', 'up', 'down']

const isAcross = (side: Direction): boolean => side === 'up' || side === 'down'

const getLeanRank = (relation: PlexRelation): number => {
  const { lean } = RELATIONS[relation]
  return lean === 'up' ? 0 : lean === null ? 1 : 2
}

function createBlocks(
  nodesByRelation: NodesByRelation,
  options: PlexOptions,
  getSize: (node: PlexNode) => Size,
): Block[] {
  return SIDES.flatMap((side): Block[] => {
    const relations = RELATION_NAMES.filter(
      (relation) =>
        options.direction[relation] === side && (nodesByRelation[relation] ?? []).length > 0,
    ).sort((first, second) => getLeanRank(first) - getLeanRank(second))
    const members = relations.flatMap((relation) => nodesByRelation[relation] ?? [])
    if (members.length === 0) return []
    const footprints = members.map(getSize)
    const cell = {
      width: Math.max(...footprints.map((footprint) => footprint.width)),
      height: Math.max(...footprints.map((footprint) => footprint.height)),
    }
    const upper = relations.filter((relation) => RELATIONS[relation].lean === 'up')
    if (isAcross(side) || upper.length === 0 || upper.length === relations.length)
      return [{ side, half: null, members, cell }]
    const upperCount = upper.flatMap((relation) => nodesByRelation[relation] ?? []).length
    return [
      { side, half: 'up', members: members.slice(0, upperCount), cell },
      { side, half: 'down', members: members.slice(upperCount), cell },
    ]
  })
}

function getBlockSize(block: Block, columns: number, grid: Grid): Size {
  const rows = Math.ceil(block.members.length / columns)
  return {
    width: columns * block.cell.width + (columns - 1) * grid.gap,
    height: rows * block.cell.height + (rows - 1) * grid.lineGap,
  }
}

function shapeAcrossBlock(block: Block, columns: number, grid: Grid): BlockShape {
  const { width, height } = getBlockSize(block, columns, grid)
  const [minY, maxY] =
    block.side === 'down'
      ? [grid.reach.y, grid.reach.y + height]
      : [-grid.reach.y - height, -grid.reach.y]
  return { columns, minX: -width / 2, maxX: width / 2, minY, maxY }
}

/** The position of a side block: centered on the row of the focus, ending over the children block, or starting under the parents block. */
type Standing = 'center' | 'up' | 'down'

const STANDINGS: readonly Standing[] = ['center', 'up', 'down']

/** The column count of every block of a side, and where a side of one block stands. */
interface SideOption {
  readonly columns: readonly number[]
  readonly standing: Standing
}

interface Span {
  readonly width: number
  readonly minY: number
  readonly maxY: number
}

function getSideSpan(block: Block, columns: number, standing: Standing, grid: Grid): Span {
  const { width, height } = getBlockSize(block, columns, grid)
  const edge = grid.reach.y - grid.lineGap
  const focusRow = block.cell.height / 2
  const [minY, maxY] =
    block.half === 'up'
      ? [focusRow - height, focusRow]
      : block.half === 'down'
        ? [focusRow + grid.lineGap, focusRow + grid.lineGap + height]
        : standing === 'up'
          ? [edge - height, edge]
          : standing === 'down'
            ? [-edge, -edge + height]
            : [-height / 2, height / 2]
  return { width, minY, maxY }
}

function getNearX({ minY, maxY }: Span, acrossShapes: readonly BlockShape[], grid: Grid): number {
  const reached = acrossShapes.filter(
    (shape) =>
      minY < shape.maxY + grid.lineGap - TOLERANCE && shape.minY - grid.lineGap + TOLERANCE < maxY,
  )
  return Math.max(grid.reach.x, ...reached.map((shape) => shape.maxX + grid.gap))
}

function shapeSideBlock(block: Block, columns: number, span: Span, near: number): BlockShape {
  const [minX, maxX] =
    block.side === 'left' ? [-near - span.width, -near] : [near, near + span.width]
  return { columns, minX, maxX, minY: span.minY, maxY: span.maxY }
}

function listFromMiddle(count: number): number[] {
  return Array.from({ length: count }, (_, index) => index).sort(
    (first, second) =>
      Math.abs(2 * first - (count - 1)) - Math.abs(2 * second - (count - 1)) || first - second,
  )
}

function listRowXs(block: Block, shape: BlockShape, count: number, grid: Grid): number[] {
  const { cell } = block
  const stepX = cell.width + grid.gap
  if (isAcross(block.side)) {
    const rowWidth = count * cell.width + (count - 1) * grid.gap
    return listFromMiddle(count).map((place) => -rowWidth / 2 + cell.width / 2 + place * stepX)
  }
  return Array.from({ length: count }, (_, column) => {
    const offset = cell.width / 2 + column * stepX
    return block.side === 'left' ? shape.maxX - offset : shape.minX + offset
  })
}

function listCells(block: Block, shape: BlockShape, grid: Grid): Position[] {
  const { cell } = block
  const count = block.members.length
  const stepY = cell.height + grid.lineGap
  const rows = Math.ceil(count / shape.columns)
  return Array.from({ length: rows }, (_, row) => {
    const inRow = Math.min(shape.columns, count - row * shape.columns)
    const y =
      block.side === 'up' || block.half === 'up'
        ? shape.maxY - cell.height / 2 - row * stepY
        : shape.minY + cell.height / 2 + row * stepY
    return listRowXs(block, shape, inRow, grid).map((x) => ({ x, y }))
  }).flat()
}

/** The notes inside the window, the columns of blocks over and under the focus beyond `ACROSS_COLUMNS`, and the distance of the notes from the focus in cells. */
interface Score {
  readonly inside: number
  readonly excess: number
  readonly distance: number
}

const NO_SCORE: Score = { inside: 0, excess: 0, distance: 0 }

function scoreCells(block: Block, cells: readonly Position[], grid: Grid): Score {
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

const addScores = (first: Score, second: Score): Score => ({
  inside: first.inside + second.inside,
  excess: first.excess + second.excess,
  distance: first.distance + second.distance,
})

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
  readonly cells: readonly Position[]
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

/** The plans of the blocks of one side, by block, column count, standing and distance from the focus. */
type SideCache = Map<string, Scored>

interface Side {
  readonly blocks: readonly Block[]
  readonly options: readonly SideOption[]
  readonly cache: SideCache
}

function createSide(blocks: readonly Block[], grid: Grid): Side {
  const columnSets = blocks.reduce<number[][]>(
    (sets, block) =>
      sets.flatMap((set) => listColumnCounts(block, grid).map((columns) => [...set, columns])),
    [[]],
  )
  const standings = blocks.length === 1 ? STANDINGS : STANDINGS.slice(0, 1)
  const options = columnSets.flatMap((columns) =>
    standings.map((standing) => ({ columns, standing })),
  )
  return { blocks, options, cache: new Map() }
}

function planSideOption(
  side: Side,
  { columns, standing }: SideOption,
  acrossShapes: readonly BlockShape[],
  grid: Grid,
): Scored[] {
  const spans = side.blocks.map((block, index) =>
    getSideSpan(block, columns[index]!, standing, grid),
  )
  const near = Math.max(...spans.map((span) => getNearX(span, acrossShapes, grid)))
  return side.blocks.map((block, index) => {
    const key = `${index}/${columns[index]}/${standing}/${near}`
    const cached = side.cache.get(key)
    if (cached) return cached
    const shape = shapeSideBlock(block, columns[index]!, spans[index]!, near)
    const cells = listCells(block, shape, grid)
    const scored = { plan: { block, cells }, shape, score: scoreCells(block, cells, grid) }
    side.cache.set(key, scored)
    return scored
  })
}

const sumScores = (scored: readonly Scored[]): Score =>
  scored.reduce((total, one) => addScores(total, one.score), NO_SCORE)

function planSide(side: Side, acrossShapes: readonly BlockShape[], grid: Grid): Scored[] {
  return side.options
    .map((option) => planSideOption(side, option, acrossShapes, grid))
    .reduce((best, next) => (isBetter(sumScores(next), sumScores(best)) ? next : best))
}

function planBlocks(blocks: readonly Block[], grid: Grid): Plan[] {
  const across = blocks.filter((block) => isAcross(block.side))
  const sides = SIDES.filter((side) => !isAcross(side))
    .map((side) => blocks.filter((block) => block.side === side))
    .filter((sideBlocks) => sideBlocks.length > 0)
    .map((sideBlocks) => createSide(sideBlocks, grid))
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
    const all = [...chosen, ...sides.flatMap((side) => planSide(side, shapes, grid))]
    const score = sumScores(all)
    if (isBetter(score, best?.score)) best = { plans: all.map((one) => one.plan), score }
  }
  choose(0, [])
  return best?.plans ?? []
}

function createGrid(focus: PlacedNode, options: PlexOptions): Grid {
  const { gap, lineGap, focusGap, viewport, margin } = options
  return {
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
  const getSize = (node: PlexNode) =>
    getFootprint(
      { width: context.width(node), height: context.options.nodeSize.height },
      context.room(node),
    )
  const blocks = createBlocks(nodesByRelation, context.options, getSize)
  if (blocks.length === 0) return []
  const grid = createGrid(focus, context.options)
  const orders = new Map(
    RELATION_NAMES.flatMap((relation) =>
      (nodesByRelation[relation] ?? []).map((node, order) => [node.id, order] as const),
    ),
  )
  const placed = new Map<string, PlacedNode>()
  for (const { block, cells } of planBlocks(blocks, grid)) {
    block.members.forEach((node, index) => {
      const cell = cells[index]!
      const footprintY = cell.y - block.cell.height / 2 + getSize(node).height / 2
      placed.set(node.id, {
        ...node,
        x: cell.x,
        y: getPillCenterY(footprintY, context.room(node)),
        width: context.width(node),
        height: context.options.nodeSize.height,
        order: orders.get(node.id) ?? 0,
        opacity: 1,
      })
    })
  }
  const ordered = RELATION_NAMES.flatMap((relation) =>
    (nodesByRelation[relation] ?? []).flatMap((node) => placed.get(node.id) ?? []),
  )
  const { shouldStop } = context
  const stopAt = shouldStop ? ordered.findIndex((node) => shouldStop(node)) : -1
  return stopAt === -1 ? ordered : ordered.slice(0, stopAt + 1)
}

export const clusters: Placement = { name: 'clusters', place: placeGrid }
