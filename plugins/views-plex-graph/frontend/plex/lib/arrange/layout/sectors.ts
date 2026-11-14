/** The sectors of the groups around the focus, and the cells of the one grid around the focus that fall in a sector. */
import { RELATIONS } from '../../relation'
import type { PlexRelation } from '../../../../types'
import type { Size } from '../options'
import type { Vector } from './spotIndex'

/** Clockwise from the top, where the parents are centered. */
export const SECTOR_ORDER: readonly PlexRelation[] = [
  'parent',
  'sibling',
  'child',
  'jump',
  'reference',
]

/** The smallest angle a group with notes gets, in degrees. */
export const MIN_SECTOR_DEGREES = 36

/** The angles of a sector in degrees; 0 points right and angles grow clockwise on screen. */
export interface Sector {
  readonly relation: PlexRelation
  readonly from: number
  readonly to: number
}

/** The direction a group grows in, in degrees. */
const HOME_DEGREES = { up: -90, down: 90, left: 180, right: 0 } as const

/**
 * Each group with notes gets the smallest angle and the rest of the circle is shared by note count. A group alone takes the
 * whole circle, centered on the direction it grows in.
 */
export function getSectors(
  counts: Readonly<Partial<Record<PlexRelation, number>>>,
  minDegrees = MIN_SECTOR_DEGREES,
): Sector[] {
  const present = SECTOR_ORDER.filter((relation) => (counts[relation] ?? 0) > 0)
  const total = present.reduce((sum, relation) => sum + (counts[relation] ?? 0), 0)
  if (total === 0) return []
  const spare = Math.max(0, 360 - minDegrees * present.length)
  const widths = present.map((relation) => minDegrees + (spare * (counts[relation] ?? 0)) / total)
  const scale = 360 / widths.reduce((sum, width) => sum + width, 0)
  const middle = present.length === 1 ? HOME_DEGREES[RELATIONS[present[0]!].grows] : -90
  let from = middle - (widths[0]! * scale) / 2
  return present.map((relation, index) => {
    const sector = { relation, from, to: from + widths[index]! * scale }
    from = sector.to
    return sector
  })
}

/** Whether the direction, in degrees, points into the sector. */
export function isInSector(angle: number, sector: Sector): boolean {
  const turned = ((((angle - sector.from) % 360) + 360) % 360) + sector.from
  return turned < sector.to
}

/** How much more a row away from the focus counts than a column away, so a group is a block wider than it is tall. */
const ROW_WEIGHT = 1.6

/**
 * How much farther a cell counts that lies behind the middle of its sector, as only a sector wider than a half circle has:
 * a lone group spreads to the sides before it goes behind the focus.
 */
const BEHIND_WEIGHT = 3

const RADIANS = Math.PI / 180

/** Without a window the grid stops growing at this many columns on either side of the focus. */
const MAX_COLUMNS = 256

export interface GridRequest {
  readonly sector: Sector
  /** The distance between two columns and between two rows. */
  readonly step: Size
  /** How far from the focus the nearest rows over and under it lie, so they keep the gap to the focus. */
  readonly firstRow: number
  /** How many cells the sector needs; without `bounds` the grid grows until it has that many. */
  readonly count: number
  /** The farthest a cell center may be from the focus, across and up or down. */
  readonly bounds?: Size | undefined
  /** Whether a cell can take a note at all; a cell that overlaps the focus cannot. */
  readonly isUsable: (cell: Vector) => boolean
}

/** A cell of the grid and how far it counts from the focus. */
export interface Cell extends Vector {
  readonly distance: number
}

/**
 * The heights of the rows up to `depth` from the focus: the row of the focus, and rows a step apart from `firstRow` on,
 * over and under it. Between the row of the focus and the first row, rows only stand a step or more from the focus row.
 */
function listRows({ step, firstRow }: GridRequest, depth: number): number[] {
  const rows = [0]
  for (let y = firstRow; y <= depth; y += step.height) rows.push(y, -y)
  for (let y = firstRow - step.height; y >= step.height; y -= step.height) rows.push(y, -y)
  return rows
}

function listCellsWithin(request: GridRequest, columns: number, depth: number): Cell[] {
  const { sector, step } = request
  const middle = ((sector.from + sector.to) / 2) * RADIANS
  const rows = listRows(request, depth)
  const cells: Cell[] = []
  for (let column = -columns; column <= columns; column++) {
    for (const y of rows) {
      if (column === 0 && y === 0) continue
      const cell = { x: column * step.width, y }
      const angle = Math.atan2(cell.y, cell.x)
      if (!isInSector(angle / RADIANS, sector) || !request.isUsable(cell)) continue
      const behind = Math.max(0, -Math.cos(angle - middle))
      cells.push({
        ...cell,
        distance: Math.hypot(cell.x, ROW_WEIGHT * cell.y) * (1 + BEHIND_WEIGHT * behind),
      })
    }
  }
  return cells
}

const byDistance = (a: Cell, b: Cell): number => a.distance - b.distance || a.y - b.y || b.x - a.x

/**
 * The cells of the grid around the focus whose centers point into the sector, nearest first. The columns lie at whole
 * steps from the focus and the rows at whole steps from the first row, so every group is laid in the same straight rows
 * and columns.
 */
export function listSectorCells(request: GridRequest): Cell[] {
  const { step, bounds } = request
  if (bounds) {
    const columns = Math.floor(bounds.width / step.width)
    return listCellsWithin(request, columns, bounds.height).sort(byDistance)
  }
  // The grid grows, and only the cells nearer than its edge are kept, so no cell left out is nearer than one kept.
  for (let columns = 4; ; columns *= 2) {
    const reach = columns * step.width
    const cells = listCellsWithin(request, columns, reach / ROW_WEIGHT).filter(
      (cell) => cell.distance <= reach,
    )
    if (cells.length >= request.count || columns >= MAX_COLUMNS) return cells.sort(byDistance)
  }
}
