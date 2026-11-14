/** The sectors of the groups around the focus, and the notes of a sector laid along one of its rings. */
import type { PlexRelation } from '../../../../types'
import type { Size } from '../options'
import { isClear, type Gaps, type Spot, type SpotIndex } from './spotIndex'

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

/** Distance in pixels between two candidate positions along a ring. */
const ARC_STEP = 3

/** A center keeps this far from the border of its sector, in pixels along the ring. */
const BORDER_PAD = 4

/** How many positions along the ring, on either side, a note tries for a clear line, in quarters of a slot. */
const LINE_TRIES = 40

const RADIANS = Math.PI / 180

/** Each group with notes gets the smallest angle and the rest of the circle is shared by note count. */
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
  let from = -90 - (widths[0]! * scale) / 2
  return present.map((relation, index) => {
    const sector = { relation, from, to: from + widths[index]! * scale }
    from = sector.to
    return sector
  })
}

export interface RingRequest {
  readonly sector: Sector
  readonly radius: number
  /** The ring is odd: its notes are shifted by half a slot. */
  readonly isOdd: boolean
  readonly gaps: Gaps
  /** The footprints of the notes still to place, in order. */
  readonly sizes: readonly Size[]
  readonly isInside: (spot: Spot) => boolean
  /** Whether the note at that position of the list keeps its lines clear at the spot; only a preference. */
  readonly isLineClear: (position: number, spot: Spot) => boolean
  /** Called for each note in turn once it has its spot; the spot is in the index by the next call. */
  readonly commit: (position: number, spot: Spot) => void
}

/** A stretch of a ring in radians, `start` before `end` clockwise. */
interface Arc {
  readonly start: number
  readonly end: number
}

interface Walk {
  readonly index: SpotIndex
  readonly request: RingRequest
  readonly arc: Arc
  readonly spots: Spot[]
}

interface Found {
  readonly spot: Spot
  readonly angle: number
}

function isFreeAt(walk: Walk, spot: Spot): boolean {
  const { request, index, spots } = walk
  return (
    request.isInside(spot) &&
    spots.every((other) => isClear(spot, other, request.gaps)) &&
    index.isFree(spot, request.gaps)
  )
}

const getSpotAt = (walk: Walk, size: Size, angle: number): Spot => ({
  x: walk.request.radius * Math.cos(angle),
  y: walk.request.radius * Math.sin(angle),
  ...size,
})

/** The first free spot for the note along the range, and the angle it is at. */
function findSpotAlong(walk: Walk, size: Size, range: Arc): Found | undefined {
  for (let angle = range.start; angle <= range.end; angle += ARC_STEP / walk.request.radius) {
    const spot = getSpotAt(walk, size, angle)
    if (isFreeAt(walk, spot)) return { spot, angle }
  }
  return undefined
}

/** The nearest free spot on either side of `found` where the lines of the note are clear. */
function findLineClearSpot(walk: Walk, position: number, found: Found): Found | undefined {
  const { request, arc } = walk
  const size = request.sizes[position]!
  const quarter = (size.width + request.gaps.gap) / 4 / request.radius
  for (let attempt = 1; attempt <= LINE_TRIES; attempt++) {
    for (const sign of [1, -1]) {
      const angle = found.angle + sign * attempt * quarter
      if (angle < arc.start || angle > arc.end) continue
      const spot = getSpotAt(walk, size, angle)
      if (isFreeAt(walk, spot) && request.isLineClear(position, spot)) return { spot, angle }
    }
  }
  return undefined
}

/** The first free spot at or after `from`, wrapping to the start of the arc, moved aside when the lines of the note are not clear there. */
function findClearSpot(walk: Walk, position: number, from: number): Found | undefined {
  const size = walk.request.sizes[position]!
  const found =
    findSpotAlong(walk, size, { start: from, end: walk.arc.end }) ??
    findSpotAlong(walk, size, { start: walk.arc.start, end: from })
  if (!found || walk.request.isLineClear(position, found.spot)) return found
  return findLineClearSpot(walk, position, found) ?? found
}

/** The notes laid one after another from `from`, as close as they fit; the list ends where a note finds no place. */
function packRing(walk: Walk, from: number): Spot[] {
  let angle = from
  for (const size of walk.request.sizes) {
    const found = findSpotAlong(walk, size, { start: angle, end: walk.arc.end })
    if (!found) break
    walk.spots.push(found.spot)
    angle = found.angle
  }
  return walk.spots
}

/** Notes spread along the arc, each at the middle of its share of it and committed as it is found; the first note that finds no place ends the ring. */
function spreadRing(walk: Walk, shift: number): void {
  const { arc, request } = walk
  const share = (arc.end - arc.start) / request.sizes.length
  for (const position of request.sizes.keys()) {
    const middle = Math.min(arc.start + (position + 0.5) * share + shift, arc.end)
    const found = findClearSpot(walk, position, middle)
    if (!found) return
    request.commit(position, found.spot)
  }
}

/**
 * Lays the notes of the request along the ring and commits them. A ring that cannot hold them all takes as many as fit, packed. A ring with
 * room to spare spreads them evenly, and alternate rings are shifted by half a slot so notes do not stand on one ray.
 */
export function layRing(index: SpotIndex, request: RingRequest): void {
  const { sector, radius, gaps, sizes } = request
  const first = sizes[0]
  if (!first) return
  const arc = {
    start: sector.from * RADIANS + BORDER_PAD / radius,
    end: sector.to * RADIANS - BORDER_PAD / radius,
  }
  const shift = request.isOdd ? (first.width + gaps.gap) / 2 / radius : 0
  const packed = packRing({ index, request, arc, spots: [] }, arc.start + shift)
  if (packed.length < sizes.length)
    packed.forEach((spot, position) => request.commit(position, spot))
  else spreadRing({ index, request, arc, spots: [] }, shift)
}
