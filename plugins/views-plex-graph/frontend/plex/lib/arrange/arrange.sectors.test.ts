/** Each group takes the cells of one even grid in its own sector around the focus, and a note is left out only when its sector has no cell left. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { getBox, isOverlapping } from './layout/boxes'
import { getSectors, MIN_SECTOR_DEGREES, SECTOR_ORDER, type Sector } from './layout/sectors'
import { DEFAULT_OPTIONS } from './options'
import { build } from '../../fixtures/build'
import type { PlexFrame } from '../frame'
import type { PlacedNode } from '../node'
import type { PlexNode, PlexRelation } from '../../../types'

const COUNTS = { parent: 3, sibling: 40, child: 40, jump: 12, reference: 8 } as const
const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const getAngle = (x: number, y: number): number => (Math.atan2(y, x) * 180) / Math.PI

const isInSector = (angle: number, sector: Sector): boolean => {
  const turned = ((((angle - sector.from) % 360) + 360) % 360) + sector.from
  return turned >= sector.from - 0.5 && turned <= sector.to + 0.5
}

const getOverlaps = (frame: PlexFrame): string[] =>
  frame.nodes.flatMap((a, index) =>
    frame.nodes
      .slice(index + 1)
      .filter((b) => isOverlapping(getBox(a, 0), getBox(b, 0)))
      .map((b) => `${a.id} / ${b.id}`),
  )

const countLeftOut = (frame: PlexFrame): number =>
  Object.values(frame.overflow).reduce((sum, count) => sum + count, 0)

/** The distinct values, each a whole number of the smallest distance between two of them from the next. */
function isEvenlySpaced(values: readonly number[]): boolean {
  const distinct = [...new Set(values.map((value) => Math.round(value * 1e6) / 1e6))].sort(
    (a, b) => a - b,
  )
  const apart = distinct.slice(1).map((value, index) => value - distinct[index]!)
  if (apart.length === 0) return true
  const step = Math.min(...apart)
  return apart.every((each) => Math.abs(each / step - Math.round(each / step)) < 1e-6)
}

/** Whether the notes stand in straight columns a step apart, and in straight rows a step apart over and under the focus. */
function isOnGrid(frame: PlexFrame): boolean {
  const shown = frame.nodes.filter((node) => node.relation !== 'focus')
  const over = shown.filter((node) => node.y < 0).map((node) => node.y)
  const under = shown.filter((node) => node.y > 0).map((node) => node.y)
  return (
    isEvenlySpaced(shown.map((node) => node.x)) && isEvenlySpaced(over) && isEvenlySpaced(under)
  )
}

const countDistinct = (values: readonly number[]): number =>
  new Set(values.map((value) => Math.round(value * 1e6))).size

/** How often the relation changes going once around the focus. */
function countRelationChanges(frame: PlexFrame): number {
  const around = frame.nodes
    .filter((node) => node.relation !== 'focus')
    .map((node) => ({ relation: node.relation, angle: getAngle(node.x, node.y) }))
    .sort((a, b) => a.angle - b.angle)
  return around.filter((node, index) => node.relation !== around.at(index - 1)!.relation).length
}

describe('the sectors', () => {
  const sectors = getSectors(COUNTS)

  it('cover the circle clockwise in the order of the groups, parents centered at the top', () => {
    expect(sectors.map((sector) => sector.relation)).toStrictEqual(SECTOR_ORDER)
    expect(sectors.at(-1)!.to - sectors[0]!.from).toBeCloseTo(360)
    expect((sectors[0]!.from + sectors[0]!.to) / 2).toBeCloseTo(-90)
    sectors.slice(1).forEach((sector, index) => expect(sector.from).toBe(sectors[index]!.to))
  })

  it('give each group with notes at least the smallest angle and more to the larger groups', () => {
    for (const sector of sectors)
      expect(sector.to - sector.from).toBeGreaterThanOrEqual(MIN_SECTOR_DEGREES - 1e-9)
    const width = (relation: PlexRelation) => {
      const sector = sectors.find((each) => each.relation === relation)!
      return sector.to - sector.from
    }
    expect(width('child')).toBeGreaterThan(width('jump'))
    expect(width('jump')).toBeGreaterThan(width('parent'))
  })

  it('leave out the groups without notes', () => {
    expect(getSectors({ child: 4, jump: 0 }).map((sector) => sector.relation)).toStrictEqual([
      'child',
    ])
  })
})

describe.each([
  { width: 1900, height: 1640 },
  { width: 1200, height: 900 },
])(
  '3 parents, 40 siblings, 40 children, 12 jumps and 8 references in $width×$height',
  (viewport) => {
    const graph = build('A note', COUNTS)

    it('show every note, each in its sector', () => {
      const frame = arrangePlex(graph, { options: { viewport } })
      const sectors = getSectors(COUNTS)

      expect(countLeftOut(frame)).toBe(0)
      expect(frame.nodes).toHaveLength(1 + 103)
      expect(getOverlaps(frame)).toStrictEqual([])
      for (const node of frame.nodes.filter((each) => each.relation !== 'focus')) {
        const sector = sectors.find((each) => each.relation === node.relation)!
        expect(isInSector(getAngle(node.x, node.y), sector), node.id).toBe(true)
      }
    })

    it('lay the notes in straight rows and columns of one grid', () => {
      const frame = arrangePlex(graph, { options: { viewport } })
      const shown = frame.nodes.filter((node) => node.relation !== 'focus')

      expect(isOnGrid(frame)).toBe(true)
      expect(countDistinct(shown.map((node) => node.x))).toBeLessThan(shown.length / 8)
      expect(countDistinct(shown.map((node) => node.y))).toBeLessThan(shown.length / 4)
    })

    it('keep each group in one run around the focus when the notes have names under them', () => {
      const frame = arrangePlex(graph, { options: { viewport, room: 20 } })

      expect(getOverlaps(frame)).toStrictEqual([])
      expect(countRelationChanges(frame)).toBe(SECTOR_ORDER.length)
    })
  },
)

it('shows every note with titles measured in 1900×1640', () => {
  const viewport = { width: 1900, height: 1640 }
  const frame = arrangePlex(build('A note', COUNTS), { options: { viewport }, measure: byTitle })

  expect(countLeftOut(frame)).toBe(0)
  expect(getOverlaps(frame)).toStrictEqual([])
})

describe('a window too small for every note', () => {
  const viewport = { width: 700, height: 500 }
  const frame = arrangePlex(build('A note', COUNTS), { options: { viewport } })

  it('leaves notes out, overlaps none and keeps each group in one run around the focus', () => {
    expect(countLeftOut(frame)).toBeGreaterThan(0)
    expect(getOverlaps(frame)).toStrictEqual([])
    expect(countRelationChanges(frame)).toBe(SECTOR_ORDER.length)
  })

  it('lays the notes it shows on one grid', () => {
    expect(isOnGrid(frame)).toBe(true)
  })

  it('shows every note of a smaller graph', () => {
    const small = arrangePlex(
      build('A note', { parent: 2, sibling: 6, child: 6, jump: 3, reference: 2 }),
      {
        options: { viewport },
      },
    )

    expect(countLeftOut(small)).toBe(0)
  })

  it('leaves notes out of a group only when its notes reach the border of the window', () => {
    const { margin, gap, lineGap } = DEFAULT_OPTIONS
    // The room a shown note leaves to the nearest border of the window.
    const getRoomToBorder = (node: PlacedNode) =>
      Math.min(
        viewport.width / 2 - margin - (Math.abs(node.x) + node.width / 2),
        viewport.height / 2 - margin - (Math.abs(node.y) + node.height / 2),
      )
    for (const relation of Object.keys(frame.overflow)) {
      const rooms = frame.nodes.filter((node) => node.relation === relation).map(getRoomToBorder)
      expect(Math.min(...rooms), relation).toBeLessThan(
        DEFAULT_OPTIONS.nodeSize.height + Math.max(gap, lineGap),
      )
    }
  })
})
