/** Where a box of a cluster goes: the free spot nearest the middle of its group. */
import type { Size } from '@numen/ui'
import type { Position } from '../../node'

export type Vector = Position

/** An upright box: its center and its size. */
export interface Spot extends Vector, Size {}

export interface Gaps {
  readonly gap: number
  readonly lineGap: number
}

/** `fits` reports whether a spot lies inside the window; a spot outside it is taken only when no spot inside is free. */
export interface SpotRequest {
  readonly start: Vector
  readonly middle: Vector
  readonly direction: Vector
  readonly size: Size
  readonly group: readonly Spot[]
  readonly gaps: Gaps
  readonly fits?: ((spot: Spot) => boolean) | undefined
  readonly isInLane?: ((point: Vector) => boolean) | undefined
  readonly isInTerritory?: ((spot: Spot) => boolean) | undefined
  readonly isLineClear?: ((spot: Spot) => boolean) | undefined
}

/**
 * Candidates are built exactly one gap from a neighbor. Without a tolerance,
 * floating-point error makes such a box count as overlapping, every touching
 * spot is refused and the groups degrade into lines.
 */
const TOLERANCE = 0.5

/** Candidates keep at least this much room from a neighbor, so with no gap they do not touch on float error. */
const LEAST_GAP = 1e-6

/** How many quarter-box steps the walk outward takes before it gives up. */
const WALK_STEPS = 400

/** Costs are compared to this many parts of a box. */
const COST_PRECISION = 1e6

/** How many spots near the group a placement tests for clear lines before it widens the search. */
const NEAR_LINE_CHECKS = 24

/** How many spots of the wide search a placement tests for clear lines before it takes the best spot whose lines were not tested. */
const WIDE_LINE_CHECKS = 160

/** The wide search looks this many half-boxes along the direction and this many cells across it, on each side. */
const WIDE_STEPS = 12
const WIDE_OFFSETS = 5

/** The offsets, in cells across the direction, tried at each step of the walk. */
const WALK_OFFSETS = [0, 1, -1, 2, -2, 3, -3] as const

/** The side of a grid cell in the index. */
const CELL = 64
const CELL_OFFSET = 1 << 20
const CELL_STRIDE = 1 << 21

export interface Rect {
  readonly minX: number
  readonly maxX: number
  readonly minY: number
  readonly maxY: number
}

/** Placed boxes, bucketed on a grid, so a spot is checked against the boxes near it and not against all of them. */
export class SpotIndex {
  private readonly cells = new Map<number, Spot[]>()
  private readonly spots: Spot[] = []
  private extent: Rect = { minX: 0, maxX: 0, minY: 0, maxY: 0 }

  constructor(spots: readonly Spot[] = []) {
    for (const spot of spots) this.add(spot)
  }

  /** Every box in the index, in the order it was added. */
  list(): readonly Spot[] {
    return this.spots
  }

  /** The bounds of every box in the index. */
  getExtent(): Rect {
    return this.extent
  }

  add(spot: Spot): void {
    this.spots.push(spot)
    this.extent = {
      minX: Math.min(this.extent.minX, spot.x - spot.width / 2),
      maxX: Math.max(this.extent.maxX, spot.x + spot.width / 2),
      minY: Math.min(this.extent.minY, spot.y - spot.height / 2),
      maxY: Math.max(this.extent.maxY, spot.y + spot.height / 2),
    }
    this.visitCells(
      {
        minX: spot.x - spot.width / 2,
        maxX: spot.x + spot.width / 2,
        minY: spot.y - spot.height / 2,
        maxY: spot.y + spot.height / 2,
      },
      (cell) => {
        const bucket = this.cells.get(cell)
        if (bucket) bucket.push(spot)
        else this.cells.set(cell, [spot])
        return false
      },
    )
  }

  /** Whether the spot keeps the gaps from every box in the index. */
  isFree(spot: Spot, gaps: Gaps): boolean {
    let free = true
    this.visitCells(
      {
        minX: spot.x - spot.width / 2 - gaps.gap,
        maxX: spot.x + spot.width / 2 + gaps.gap,
        minY: spot.y - spot.height / 2 - gaps.lineGap,
        maxY: spot.y + spot.height / 2 + gaps.lineGap,
      },
      (cell) => {
        for (const other of this.cells.get(cell) ?? []) {
          if (!isClear(spot, other, gaps)) {
            free = false
            return true
          }
        }
        return false
      },
    )
    return free
  }

  /** The boxes in the cells the rectangle touches, so a few more than the ones it overlaps. */
  listNear(rect: Rect): Spot[] {
    const near = new Set<Spot>()
    this.visitCells(rect, (cell) => {
      for (const spot of this.cells.get(cell) ?? []) near.add(spot)
      return false
    })
    return [...near]
  }

  private visitCells(rect: Rect, visit: (cell: number) => boolean): void {
    const toX = Math.floor(rect.maxX / CELL)
    const fromY = Math.floor(rect.minY / CELL)
    const toY = Math.floor(rect.maxY / CELL)
    for (let x = Math.floor(rect.minX / CELL); x <= toX; x++) {
      for (let y = fromY; y <= toY; y++) {
        if (visit((x + CELL_OFFSET) * CELL_STRIDE + y + CELL_OFFSET)) return
      }
    }
  }
}

interface Rank {
  readonly cost: number
  readonly progress: number
  readonly pull: number
}

function isBetter(rank: Rank, best: Rank): boolean {
  if (rank.cost !== best.cost) return rank.cost < best.cost
  const progress = roundProgress(rank.progress)
  const bestProgress = roundProgress(best.progress)
  return progress < bestProgress || (progress === bestProgress && rank.pull < best.pull)
}

/** How far the center of the group would lie from its middle with a box at the point, in cells. */
export function measurePull(point: Vector, request: SpotRequest): number {
  const { middle, size, gaps, group } = request
  let sumX = point.x
  let sumY = point.y
  for (const box of group) {
    sumX += box.x
    sumY += box.y
  }
  const count = group.length + 1
  return Math.round(
    Math.hypot(
      (sumX / count - middle.x) / (size.width + gaps.gap),
      (sumY / count - middle.y) / (size.height + gaps.lineGap),
    ) * COST_PRECISION,
  )
}

function roundProgress(progress: number): number {
  return Math.round(progress * COST_PRECISION)
}

function getRequiredGap(gap: number): number {
  return Math.max(0, gap - TOLERANCE)
}

function isClear(spot: Spot, other: Spot, gaps: Gaps): boolean {
  return (
    Math.abs(spot.x - other.x) >= (spot.width + other.width) / 2 + getRequiredGap(gaps.gap) ||
    Math.abs(spot.y - other.y) >= (spot.height + other.height) / 2 + getRequiredGap(gaps.lineGap)
  )
}

/**
 * The free spot of least cost: the nearest to the middle, with costs rounded
 * before they are compared so floating-point error does not decide a tie. Of
 * equal costs the one nearer the start wins, which fills a group across its
 * axis before it goes deeper; of those the one that keeps the center of the
 * group nearest its middle wins, so a group grows on both sides of its axis in
 * turn; of those the earlier candidate wins.
 *
 * The lane of the group is searched first, and in it a spot whose lines are
 * clear before one whose lines are not. The rest of the window is searched
 * only when the lane has no free spot.
 */
export function findFreeSpot(index: SpotIndex, request: SpotRequest): Spot {
  return findSpotWithin(index, request) ?? findSpotBeyond(index, request)
}

/** The spot of `findFreeSpot`, or nothing when the window and the territory of the group hold no free spot. */
export function findSpotWithin(index: SpotIndex, request: SpotRequest): Spot | undefined {
  const inLane = request.isInLane ? [request] : []
  const tiers = [...inLane, { ...request, isInLane: undefined }]
  const plains: Spot[] = []
  for (const tier of tiers) {
    const plain = searchSpot(index, tier)
    if (!plain) continue
    if (!tier.isLineClear || tier.isLineClear(plain)) return plain
    const clear =
      searchSpot(index, tier, { left: NEAR_LINE_CHECKS }) ??
      findClearSpotWide(index, tier, { left: WIDE_LINE_CHECKS })
    if (clear) return clear
    plains.push(plain)
  }
  return plains[0]
}

function isAllowed(spot: Spot, request: SpotRequest): boolean {
  return (
    (!request.fits || request.fits(spot)) && (!request.isInTerritory || request.isInTerritory(spot))
  )
}

function searchSpot(
  index: SpotIndex,
  request: SpotRequest,
  budget?: { left: number },
): Spot | undefined {
  const { size, gaps } = request
  const isLineOk = (spot: Spot): boolean =>
    !budget || !request.isLineClear || (budget.left-- > 0 && request.isLineClear(spot))
  const toSpot = (point: Vector): Spot => ({ x: point.x, y: point.y, ...size })

  const found: { best?: Rank & { point: Vector } } = {}
  const consider = (point: Vector, minProgress: number) => {
    const progress = measureProgress(point, request)
    if (progress < minProgress) return
    if (request.isInLane && !request.isInLane(point)) return
    const cost = Math.round(measureCost(point, request) * COST_PRECISION)
    const rank = { cost, progress, pull: measurePull(point, request) }
    if (found.best && !isBetter(rank, found.best)) return
    if (!isAllowed(toSpot(point), request)) return
    if (index.isFree(toSpot(point), gaps) && isLineOk(toSpot(point))) {
      found.best = { point, ...rank }
    }
  }
  const considerAll = (points: readonly Vector[], minProgress: number) =>
    points.forEach((point) => consider(point, minProgress))

  considerAll([request.middle, request.start], -TOLERANCE)
  for (const box of request.group) {
    // The candidates of a box that all cost more than the best so far cannot win, so they are not built.
    if (found.best && getLeastCost(box, request) > found.best.cost + 1) continue
    considerAll(listNearbyCandidates(box, request), -TOLERANCE)
  }
  if (!found.best && request.fits) {
    // The group has no free place in the window on its side of the start. A box wider than the ones before it starts further out, so a place
    // that is free and in the window is also taken where the box reaches back past the start, as long as its center is on the side of the group.
    const relaxedMinProgress = -getReach(request.direction, request.size) - TOLERANCE
    for (const box of index.list())
      considerAll(listNearbyCandidates(box, request), relaxedMinProgress)
  }
  if (found.best) return toSpot(found.best.point)
  return budget ? undefined : findFreeSpotOutward(index, request)
}

function listWideSpots(request: SpotRequest): Spot[] {
  const { middle, direction, size, gaps } = request
  const cell = { width: size.width + gaps.gap, height: size.height + gaps.lineGap }
  const stride = size.height / 2
  const spots: Spot[] = []
  for (let step = 0; step <= WIDE_STEPS; step++) {
    for (let offset = -WIDE_OFFSETS; offset <= WIDE_OFFSETS; offset++) {
      const spot: Spot = {
        x: middle.x + direction.x * step * stride - direction.y * offset * cell.width,
        y: middle.y + direction.y * step * stride + direction.x * offset * cell.height,
        ...size,
      }
      const isOwn = !request.isInLane || request.isInLane(spot)
      const isAllowed = !request.isInTerritory || request.isInTerritory(spot)
      if (isOwn && isAllowed && measureProgress(spot, request) >= -TOLERANCE) spots.push(spot)
    }
  }
  return spots.sort(
    (first, second) =>
      measureCost(first, request) - measureCost(second, request) ||
      measurePull(first, request) - measurePull(second, request),
  )
}

function findClearSpotWide(
  index: SpotIndex,
  request: SpotRequest,
  budget: { left: number },
): Spot | undefined {
  for (const spot of listWideSpots(request)) {
    if (budget.left <= 0) return undefined
    if (request.fits && !request.fits(spot)) continue
    if (!index.isFree(spot, request.gaps)) continue
    budget.left--
    if (request.isLineClear?.(spot)) return spot
  }
  return undefined
}

function findSpotBeyond(index: SpotIndex, request: SpotRequest): Spot {
  const { middle, direction, size, gaps } = request
  const { minX, maxX, minY, maxY } = index.getExtent()
  if (Math.abs(direction.x) >= Math.abs(direction.y)) {
    const x =
      direction.x >= 0 ? maxX + gaps.gap + size.width / 2 + 1 : minX - gaps.gap - size.width / 2 - 1
    return { x, y: middle.y, ...size }
  }
  const y =
    direction.y >= 0
      ? maxY + gaps.lineGap + size.height / 2 + 1
      : minY - gaps.lineGap - size.height / 2 - 1
  return { x: middle.x, y, ...size }
}

function getCandidateGaps(request: SpotRequest): Gaps {
  return {
    gap: Math.max(request.gaps.gap, LEAST_GAP),
    lineGap: Math.max(request.gaps.lineGap, LEAST_GAP),
  }
}

function getLeastCost(box: Spot, request: SpotRequest): number {
  const { middle, size } = request
  const gaps = getCandidateGaps(request)
  const reachX = box.width / 2 + gaps.gap + size.width / 2
  const reachY = box.height / 2 + gaps.lineGap + size.height / 2
  const farX = Math.max(0, Math.abs(middle.x - box.x) - reachX)
  const farY = Math.max(0, Math.abs(middle.y - box.y) - reachY)
  const least = Math.hypot(
    farX / (size.width + request.gaps.gap),
    farY / (size.height + request.gaps.lineGap),
  )
  return least * COST_PRECISION
}

function getReach(direction: Vector, box: Size): number {
  return Math.abs(direction.x) * (box.width / 2) + Math.abs(direction.y) * (box.height / 2)
}

export function listNearbyCandidates(box: Spot, request: SpotRequest): Vector[] {
  const { size } = request
  const gaps = getCandidateGaps(request)
  const right = box.x + box.width / 2 + gaps.gap + size.width / 2
  const left = box.x - box.width / 2 - gaps.gap - size.width / 2
  const below = box.y + box.height / 2 + gaps.lineGap + size.height / 2
  const above = box.y - box.height / 2 - gaps.lineGap - size.height / 2
  return [
    { x: right, y: box.y },
    { x: left, y: box.y },
    { x: box.x, y: below },
    { x: box.x, y: above },
    { x: (box.x + right) / 2, y: below },
    { x: (box.x + left) / 2, y: below },
    { x: (box.x + right) / 2, y: above },
    { x: (box.x + left) / 2, y: above },
  ]
}

export function measureProgress(point: Vector, request: SpotRequest): number {
  const { start, direction } = request
  return (point.x - start.x) * direction.x + (point.y - start.y) * direction.y
}

export function measureCost(point: Vector, request: SpotRequest): number {
  const { middle, size, gaps } = request
  return Math.hypot(
    (point.x - middle.x) / (size.width + gaps.gap),
    (point.y - middle.y) / (size.height + gaps.lineGap),
  )
}

function findFreeSpotOutward(index: SpotIndex, request: SpotRequest): Spot | undefined {
  const { middle, direction, size, gaps } = request
  const cell = { width: size.width + gaps.gap, height: size.height + gaps.lineGap }
  const stride = size.height / 4

  for (let step = 1; step <= WALK_STEPS; step++) {
    for (const offset of WALK_OFFSETS) {
      const spot: Spot = {
        x: middle.x + direction.x * step * stride - direction.y * offset * cell.width,
        y: middle.y + direction.y * step * stride + direction.x * offset * cell.height,
        ...size,
      }
      if (!isAllowed(spot, request)) continue
      if (request.isInLane && !request.isInLane(spot)) continue
      if (!index.isFree(spot, gaps) || measureProgress(spot, request) < -TOLERANCE) continue
      return spot
    }
  }
  return undefined
}
