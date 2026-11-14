/** Where the lines between placed notes run, so a note is not set where its line would hide another line or run through another note. */
import type { Rect, Spot, SpotIndex, Vector } from './freeSpot'

export interface Link {
  readonly from: string
  readonly to: string
}

/** Two lines closer than this (the sine of the angle between them) to parallel count as one line. */
const PARALLEL_SINE = 0.12

/** How far a line keeps from the box of a note it does not end at: room for a title to sit beside it. */
const CLEARANCE = 24

const CELL = 64
const CELL_OFFSET = 1 << 20
const CELL_STRIDE = 1 << 21

interface Segment {
  readonly from: Vector
  readonly to: Vector
}

const getBounds = (segment: Segment): Rect => ({
  minX: Math.min(segment.from.x, segment.to.x),
  maxX: Math.max(segment.from.x, segment.to.x),
  minY: Math.min(segment.from.y, segment.to.y),
  maxY: Math.max(segment.from.y, segment.to.y),
})

const grow = (rect: Rect, by: number): Rect => ({
  minX: rect.minX - by,
  maxX: rect.maxX + by,
  minY: rect.minY - by,
  maxY: rect.maxY + by,
})

const getBoxBounds = (box: Spot): Rect => ({
  minX: box.x - box.width / 2,
  maxX: box.x + box.width / 2,
  minY: box.y - box.height / 2,
  maxY: box.y + box.height / 2,
})

function isParallel(first: Segment, second: Segment): boolean {
  const ux = first.to.x - first.from.x
  const uy = first.to.y - first.from.y
  const vx = second.to.x - second.from.x
  const vy = second.to.y - second.from.y
  return Math.abs(ux * vy - uy * vx) < PARALLEL_SINE * Math.hypot(ux, uy) * Math.hypot(vx, vy)
}

function isCrossing(segment: Segment, rect: Rect): boolean {
  const dx = segment.to.x - segment.from.x
  const dy = segment.to.y - segment.from.y
  const edges: readonly (readonly [number, number])[] = [
    [-dx, segment.from.x - rect.minX],
    [dx, rect.maxX - segment.from.x],
    [-dy, segment.from.y - rect.minY],
    [dy, rect.maxY - segment.from.y],
  ]
  let enter = 0
  let leave = 1
  for (const [direction, distance] of edges) {
    if (direction === 0) {
      if (distance < 0) return false
      continue
    }
    const at = distance / direction
    if (direction < 0) enter = Math.max(enter, at)
    else leave = Math.min(leave, at)
    if (enter > leave) return false
  }
  return true
}

function listCells(rect: Rect): number[] {
  const cells: number[] = []
  const toX = Math.floor(rect.maxX / CELL)
  const toY = Math.floor(rect.maxY / CELL)
  for (let x = Math.floor(rect.minX / CELL); x <= toX; x++) {
    for (let y = Math.floor(rect.minY / CELL); y <= toY; y++) {
      cells.push((x + CELL_OFFSET) * CELL_STRIDE + y + CELL_OFFSET)
    }
  }
  return cells
}

/**
 * The lines between placed notes, drawn straight from center to center. A line that meets another line at a
 * note is not set along it, except at the hub, where a line on each side is how a parent and a child sit.
 */
export class LineIndex {
  private readonly neighbors = new Map<string, string[]>()
  private readonly centers = new Map<string, Vector>()
  private readonly boxes = new Map<string, Spot>()
  private readonly cells = new Map<number, Segment[]>()

  private readonly hub: string
  private readonly boxIndex: SpotIndex

  constructor(links: readonly Link[], hub: string, boxIndex: SpotIndex) {
    this.hub = hub
    this.boxIndex = boxIndex
    for (const { from, to } of links) {
      if (from === to) continue
      this.link(from, to)
      this.link(to, from)
    }
  }

  /** Records a placed note; the box is the one that went into the box index. */
  place(id: string, center: Vector, box: Spot): void {
    for (const other of this.listPlacedNeighbors(id)) {
      const segment = { from: center, to: this.centers.get(other)! }
      for (const cell of listCells(getBounds(segment))) {
        const bucket = this.cells.get(cell)
        if (bucket) bucket.push(segment)
        else this.cells.set(cell, [segment])
      }
    }
    this.centers.set(id, center)
    this.boxes.set(id, box)
  }

  /** Whether a note at the spot, with its pill centered on `center`, keeps its lines apart from the other lines and notes. */
  isClear(id: string, center: Vector, box: Spot): boolean {
    const own = this.listPlacedNeighbors(id).map((other) => ({
      other,
      segment: { from: center, to: this.centers.get(other)! },
    }))
    return (
      !this.isCrossed(box) &&
      own.every(({ other, segment }) => !this.isBlocked(segment, other)) &&
      own.every(({ other, segment }, index) => this.isSeparate(segment, other, own, index))
    )
  }

  private link(from: string, to: string): void {
    const list = this.neighbors.get(from)
    if (list) list.push(to)
    else this.neighbors.set(from, [to])
  }

  private listPlacedNeighbors(id: string): string[] {
    return (this.neighbors.get(id) ?? []).filter((other) => this.centers.has(other))
  }

  private isCrossed(box: Spot): boolean {
    const rect = grow(getBoxBounds(box), CLEARANCE)
    for (const cell of listCells(rect)) {
      for (const segment of this.cells.get(cell) ?? []) if (isCrossing(segment, rect)) return true
    }
    return false
  }

  private isBlocked(segment: Segment, other: string): boolean {
    const own = this.boxes.get(other)
    const area = grow(getBounds(segment), CLEARANCE)
    return this.boxIndex
      .listNear(area)
      .some((box) => box !== own && isCrossing(segment, grow(getBoxBounds(box), CLEARANCE)))
  }

  private isSeparate(
    segment: Segment,
    other: string,
    own: readonly { other: string; segment: Segment }[],
    index: number,
  ): boolean {
    for (const sibling of own.slice(index + 1))
      if (isParallel(segment, sibling.segment)) return false
    if (other === this.hub) return true
    const middle = this.centers.get(other)!
    for (const third of this.listPlacedNeighbors(other)) {
      if (isParallel(segment, { from: middle, to: this.centers.get(third)! })) return false
    }
    return true
  }
}
