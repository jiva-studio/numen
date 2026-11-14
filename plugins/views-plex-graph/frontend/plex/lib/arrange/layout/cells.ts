/**
 * The cells of a block of the grid: how large a block is at a column count, where its cells lie, and how a layout of
 * blocks scores against the window.
 */
import type { PlacedNode, Position } from '../../node'
import type { Lean } from '../../relation'
import type { Direction, PlexOptions, Size } from '../options'
import type { PlexNode } from '../../../../types'

/** The notes of one side, or of its upper or lower half, in the order they fill the cells, and the size of every cell. */
export interface Block {
  readonly side: Direction
  readonly half: Lean | null
  readonly members: readonly PlexNode[]
  readonly cell: Size
}

export interface BlockShape {
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
export interface Grid {
  readonly gap: number
  readonly lineGap: number
  readonly reach: { readonly x: number; readonly y: number }
  readonly window: { readonly x: number; readonly y: number }
}

/** The least gap between two neighbors, so neighbors with no gap do not overlap on float error. */
const LEAST_GAP = 1e-6

export const isAcross = (side: Direction): boolean => side === 'up' || side === 'down'

export function getBlockSize(block: Block, columns: number, grid: Grid): Size {
  const rows = Math.ceil(block.members.length / columns)
  return {
    width: columns * block.cell.width + (columns - 1) * grid.gap,
    height: rows * block.cell.height + (rows - 1) * grid.lineGap,
  }
}

const middleOrders = new Map<number, readonly number[]>()

function listFromMiddle(count: number): readonly number[] {
  const cached = middleOrders.get(count)
  if (cached) return cached
  const order = Array.from({ length: count }, (_, index) => index).sort(
    (first, second) =>
      Math.abs(2 * first - (count - 1)) - Math.abs(2 * second - (count - 1)) || first - second,
  )
  middleOrders.set(count, order)
  return order
}

function forEachCell(
  block: Block,
  shape: BlockShape,
  grid: Grid,
  visit: (x: number, y: number) => void,
): void {
  const { cell } = block
  const count = block.members.length
  const stepX = cell.width + grid.gap
  const stepY = cell.height + grid.lineGap
  const rows = Math.ceil(count / shape.columns)
  const isGrowingUp = block.side === 'up' || block.half === 'up'
  for (let row = 0; row < rows; row++) {
    const inRow = Math.min(shape.columns, count - row * shape.columns)
    const y = isGrowingUp
      ? shape.maxY - cell.height / 2 - row * stepY
      : shape.minY + cell.height / 2 + row * stepY
    if (isAcross(block.side)) {
      const rowWidth = inRow * cell.width + (inRow - 1) * grid.gap
      for (const place of listFromMiddle(inRow))
        visit(-rowWidth / 2 + cell.width / 2 + place * stepX, y)
      continue
    }
    for (let column = 0; column < inRow; column++) {
      const offset = cell.width / 2 + column * stepX
      visit(block.side === 'left' ? shape.maxX - offset : shape.minX + offset, y)
    }
  }
}

export function listCells(block: Block, shape: BlockShape, grid: Grid): Position[] {
  const cells: Position[] = []
  forEachCell(block, shape, grid, (x, y) => cells.push({ x, y }))
  return cells
}

/**
 * The notes inside the window, the columns of blocks over and under the focus beyond `ACROSS_COLUMNS` of clusters.ts, the sides that
 * split and are laid as one block, and the distance of the notes from the focus in cells.
 */
export interface Score {
  readonly inside: number
  readonly excess: number
  readonly joined: number
  readonly distance: number
}

const NO_SCORE: Score = { inside: 0, excess: 0, joined: 0, distance: 0 }

export function scoreBlock(block: Block, shape: BlockShape, grid: Grid): Score {
  const { cell } = block
  const { window } = grid
  const stepX = cell.width + grid.gap
  const stepY = cell.height + grid.lineGap
  let inside = 0
  let distance = 0
  forEachCell(block, shape, grid, (x, y) => {
    if (Math.abs(x) + cell.width / 2 <= window.x && Math.abs(y) + cell.height / 2 <= window.y)
      inside++
    distance += Math.hypot(x / stepX, y / stepY)
  })
  return { inside, excess: 0, joined: 0, distance }
}

export const isBetter = (score: Score, best: Score | undefined): boolean =>
  !best ||
  score.inside > best.inside ||
  (score.inside === best.inside &&
    (score.excess < best.excess ||
      (score.excess === best.excess &&
        (score.joined < best.joined ||
          (score.joined === best.joined && score.distance < best.distance - 1e-9)))))

const addScores = (first: Score, second: Score): Score => ({
  inside: first.inside + second.inside,
  excess: first.excess + second.excess,
  joined: first.joined + second.joined,
  distance: first.distance + second.distance,
})

export const sumScores = (scores: readonly Score[]): Score => scores.reduce(addScores, NO_SCORE)

function getMaxColumns(block: Block, grid: Grid): number {
  const { gap, reach, window } = grid
  const room = isAcross(block.side) ? 2 * window.x : window.x - reach.x
  const fitting = Math.floor((room + gap) / (block.cell.width + gap))
  const square = Math.ceil(Math.sqrt(block.members.length)) * 2
  const limit = Number.isFinite(fitting) ? Math.max(1, fitting) : square
  return Math.max(1, Math.min(block.members.length, limit))
}

export function listColumnCounts(block: Block, grid: Grid): number[] {
  return Array.from({ length: getMaxColumns(block, grid) }, (_, index) => index + 1)
}

export function createGrid(focus: PlacedNode, options: PlexOptions): Grid {
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
