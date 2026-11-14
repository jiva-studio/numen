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
 * rows reach, and its rows fill from the focus out. When the window cuts notes, the placement is asked to fit first
 * (`PlacementContext.isFitFirst`): a side of two blocks may then be laid as one block, its references over its jumps,
 * if that block holds the whole side inside the window.
 *
 * The column counts and the side block positions are chosen together: the most notes inside the window first, then the
 * fewest columns a block over or under the focus takes beyond `ACROSS_COLUMNS`, then the fewest sides of two blocks
 * laid as one, then the least distance of the notes from the focus, counted in cells. A block takes at most as many columns as reach the edge of the window; with no
 * window, twice the square root of its note count, rounded up.
 */
import { RELATION_NAMES, RELATIONS } from '../../relation'
import type { PlacedNode, Position } from '../../node'
import type { Direction, PlexOptions, Size } from '../options'
import {
  createGrid,
  getBlockSize,
  isAcross,
  isBetter,
  listCells,
  listColumnCounts,
  scoreBlock,
  sumScores,
  type Block,
  type BlockShape,
  type Grid,
  type Score,
} from './cells'
import {
  getFootprint,
  getPillCenterY,
  type NodesByRelation,
  type Placement,
  type PlacementContext,
} from './placement'
import type { PlexNode, PlexRelation } from '../../../../types'

/** Blocks that touch within this distance do not count as reaching each other. */
const TOLERANCE = 1e-6

/** The column count of a block over or under the focus past which it takes more columns only to show more notes. */
const ACROSS_COLUMNS = 5

const SIDES: readonly Direction[] = ['left', 'right', 'up', 'down']

const getLeanRank = (relation: PlexRelation): number => {
  const { lean } = RELATIONS[relation]
  return lean === 'up' ? 0 : lean === null ? 1 : 2
}

/** The blocks of one side: its one block, and for a side of two blocks also its two halves. */
interface SideBlocks {
  readonly whole: Block
  readonly halves?: readonly [upper: Block, lower: Block]
}

function createBlocks(
  nodesByRelation: NodesByRelation,
  options: PlexOptions,
  getSize: (node: PlexNode) => Size,
): SideBlocks[] {
  return SIDES.flatMap((side): SideBlocks[] => {
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
    const whole: Block = { side, half: null, members, cell }
    const upper = relations.filter((relation) => RELATIONS[relation].lean === 'up')
    if (isAcross(side) || upper.length === 0 || upper.length === relations.length)
      return [{ whole }]
    const upperCount = upper.flatMap((relation) => nodesByRelation[relation] ?? []).length
    const halves = [
      { side, half: 'up', members: members.slice(0, upperCount), cell },
      { side, half: 'down', members: members.slice(upperCount), cell },
    ] as const
    return [{ whole, halves }]
  })
}

function shapeAcrossBlock(block: Block, columns: number, grid: Grid): BlockShape {
  const { width, height } = getBlockSize(block, columns, grid)
  const [minY, maxY] =
    block.side === 'down'
      ? [grid.reach.y, grid.reach.y + height]
      : [-grid.reach.y - height, -grid.reach.y]
  return { columns, minX: -width / 2, maxX: width / 2, minY, maxY }
}

/** The position of a side block of one block: centered on the row of the focus, ending over the children block, or starting under the parents block. */
type Standing = 'center' | 'up' | 'down'

const STANDINGS: readonly Standing[] = ['center', 'up', 'down']

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
  let near = grid.reach.x
  for (const shape of acrossShapes) {
    const isReached =
      minY < shape.maxY + grid.lineGap - TOLERANCE && shape.minY - grid.lineGap + TOLERANCE < maxY
    if (isReached) near = Math.max(near, shape.maxX + grid.gap)
  }
  return near
}

function shapeSideBlock(block: Block, columns: number, span: Span, near: number): BlockShape {
  const [minX, maxX] =
    block.side === 'left' ? [-near - span.width, -near] : [near, near + span.width]
  return { columns, minX, maxX, minY: span.minY, maxY: span.maxY }
}

interface Plan {
  readonly block: Block
  readonly cells: readonly Position[]
}

interface AcrossChoice {
  readonly columns: number
  readonly block: Block
  readonly shape: BlockShape
  readonly score: Score
}

function chooseAcross(block: Block, columns: number, grid: Grid): AcrossChoice {
  const shape = shapeAcrossBlock(block, columns, grid)
  const score = { ...scoreBlock(block, shape, grid), excess: Math.max(0, columns - ACROSS_COLUMNS) }
  return { columns, block, shape, score }
}

/**
 * The scores of side blocks by the distance of their near column from the focus, kept for the probes that lay the
 * same blocks at the same options again: each block of a side, its column count and placing.
 */
type ScoreCache = Map<string, Map<number, Score>>

const scoreCaches = new WeakMap<PlexOptions, ScoreCache>()

/** One block of a side at one column count and placing, and its scores by the distance of its near column. */
interface SidePart {
  readonly block: Block
  readonly columns: number
  readonly span: Span
  readonly scores: Map<number, Score>
}

/** The parts of a side laid together, and whether they are the one block of a side that splits. */
interface SideOption {
  readonly parts: readonly SidePart[]
  readonly joined: number
}

function createPart(
  block: Block,
  columns: number,
  standing: Standing,
  { grid, cache }: { readonly grid: Grid; readonly cache: ScoreCache },
): SidePart {
  const { side, half, members, cell } = block
  const key = `${side}/${half}/${members.length}/${cell.width}/${cell.height}/${columns}/${standing}/${grid.reach.x}/${grid.reach.y}`
  let scores = cache.get(key)
  if (!scores) {
    scores = new Map()
    cache.set(key, scores)
  }
  return { block, columns, span: getSideSpan(block, columns, standing, grid), scores }
}

function createSide(
  { whole, halves }: SideBlocks,
  { grid, cache, isFitFirst }: { grid: Grid; cache: ScoreCache; isFitFirst: boolean },
): SideOption[] {
  const context = { grid, cache }
  const split = halves
    ? listColumnCounts(halves[0], grid).flatMap((upperColumns) =>
        listColumnCounts(halves[1], grid).map((lowerColumns) => ({
          parts: [
            createPart(halves[0], upperColumns, 'center', context),
            createPart(halves[1], lowerColumns, 'center', context),
          ],
          joined: 0,
        })),
      )
    : []
  if (halves && !isFitFirst) return split
  const joined = listColumnCounts(whole, grid).flatMap((columns) =>
    STANDINGS.map((standing) => ({
      parts: [createPart(whole, columns, standing, context)],
      joined: halves ? 1 : 0,
    })),
  )
  return [...split, ...joined]
}

function scorePart(part: SidePart, near: number, grid: Grid): Score {
  const cached = part.scores.get(near)
  if (cached) return cached
  const score = scoreBlock(
    part.block,
    shapeSideBlock(part.block, part.columns, part.span, near),
    grid,
  )
  part.scores.set(near, score)
  return score
}

interface SideChoice {
  readonly option: SideOption
  readonly near: number
  readonly scores: readonly Score[]
  readonly score: Score
}

function chooseSide(
  options: readonly SideOption[],
  acrossShapes: readonly BlockShape[],
  grid: Grid,
): SideChoice | undefined {
  let best: SideChoice | undefined
  for (const option of options) {
    let near = grid.reach.x
    for (const part of option.parts) near = Math.max(near, getNearX(part.span, acrossShapes, grid))
    const scores = option.parts.map((part) => scorePart(part, near, grid))
    const score = { ...sumScores(scores), joined: option.joined }
    // A cell inside holds its note, while a note can lie inside a cell that is not: one block that holds less than all
    // of the side could hold fewer notes than the halves do.
    if (option.joined > 0 && score.inside < option.parts[0]!.block.members.length) continue
    if (isBetter(score, best?.score)) best = { option, near, scores, score }
  }
  return best
}

function planBlocks(
  blocks: readonly SideBlocks[],
  context: { grid: Grid; cache: ScoreCache; isFitFirst: boolean },
): Plan[] {
  const { grid } = context
  const across = blocks.flatMap(({ whole }) => (isAcross(whole.side) ? [whole] : []))
  const sides = blocks
    .filter(({ whole }) => !isAcross(whole.side))
    .map((side) => createSide(side, context))
  // The column counts are tried best first, so the bound below leaves out most of them.
  const acrossChoices = across.map((block) =>
    listColumnCounts(block, grid)
      .map((columns) => chooseAcross(block, columns, grid))
      .sort((first, second) =>
        isBetter(first.score, second.score) ? -1 : isBetter(second.score, first.score) ? 1 : 0,
      ),
  )
  let best: { across: AcrossChoice[]; sides: SideChoice[]; score: Score } | undefined
  // Of two layouts that score the same, the one with fewer columns in the first block over or under the focus wins, then in the next.
  const isEarlier = (chosen: readonly AcrossChoice[], other: readonly AcrossChoice[]): boolean => {
    const index = chosen.findIndex((one, at) => one.columns !== other[at]!.columns)
    return index !== -1 && chosen[index]!.columns < other[index]!.columns
  }
  const isWinning = (score: Score, chosen: readonly AcrossChoice[]): boolean =>
    !best ||
    isBetter(score, best.score) ||
    (!isBetter(best.score, score) && isEarlier(chosen, best.across))
  // A side scores best with its blocks nearest the focus, where no block over or under the focus pushes them out:
  // a set of column counts whose score with that best is not winning is left out.
  const sideBounds = sides.flatMap((options) => chooseSide(options, [], grid)?.score ?? [])
  const choose = (index: number, chosen: readonly AcrossChoice[]) => {
    if (index < across.length) {
      for (const option of acrossChoices[index]!) choose(index + 1, [...chosen, option])
      return
    }
    const bound = sumScores([...chosen.map((one) => one.score), ...sideBounds])
    if (!isWinning(bound, chosen)) return
    const shapes = chosen.map((one) => one.shape)
    const sideChoices = sides.flatMap((options) => chooseSide(options, shapes, grid) ?? [])
    const scores = [
      ...chosen.map((one) => one.score),
      ...sideChoices.flatMap((choice) => choice.scores),
    ]
    const joined = sideChoices.reduce((total, choice) => total + choice.option.joined, 0)
    const score = { ...sumScores(scores), joined }
    if (isWinning(score, chosen)) best = { across: [...chosen], sides: sideChoices, score }
  }
  choose(0, [])
  if (!best) return []
  return [
    ...best.across.map(({ block, shape }) => ({ block, cells: listCells(block, shape, grid) })),
    ...best.sides.flatMap(({ option, near }) =>
      option.parts.map(({ block, columns, span }) => ({
        block,
        cells: listCells(block, shapeSideBlock(block, columns, span, near), grid),
      })),
    ),
  ]
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
  let cache = scoreCaches.get(context.options)
  if (!cache) {
    cache = new Map()
    scoreCaches.set(context.options, cache)
  }
  const plans = planBlocks(blocks, { grid, cache, isFitFirst: context.isFitFirst === true })
  const placed = new Map<string, PlacedNode>()
  for (const { block, cells } of plans) {
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
