/** The look settings, as the arrangement turns them into boxes and gaps. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { resolveOptions, type PlexOptionsInput } from './options'
import { optionsForType, DESIGNED_TYPE } from '../../model/sizing/plexScale'
import { graphs } from '../../fixtures/graphs'
import { build } from '../../fixtures/build'
import type { PlacedNode } from '../node'
import type { PlexFrame } from '../frame'
import type { PlexGraph, PlexNode } from '../../../types'

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const LARGE = { width: 1200, height: 800 }

/**
 * The boxes of the typical graph as the plex drew them before the look settings,
 * with titles measured by `byTitle`: `[id, width, height]`. Their position is the
 * placement's business and is not pinned here.
 */
const TODAY: readonly (readonly [string, number, number])[] = [
  ['focus', 176, 44],
  ['parent-0', 144, 36],
  ['parent-1', 144, 36],
  ['child-0', 82, 36],
  ['child-1', 72, 36],
  ['child-2', 89, 36],
  ['child-3', 144, 36],
  ['child-4', 96, 36],
  ['child-5', 103, 36],
  ['jump-0', 144, 36],
  ['jump-1', 144, 36],
  ['jump-2', 144, 36],
  ['sibling-0', 144, 36],
  ['sibling-1', 144, 36],
]

const TODAY_EDGES = [
  ['parent-0', 'focus'],
  ['parent-1', 'focus'],
  ['focus', 'child-0'],
  ['focus', 'child-1'],
  ['focus', 'child-2'],
  ['focus', 'child-3'],
  ['focus', 'child-4'],
  ['focus', 'child-5'],
  ['jump-0', 'focus'],
  ['jump-1', 'focus'],
  ['jump-2', 'focus'],
  ['parent-0', 'sibling-0'],
  ['parent-0', 'sibling-1'],
]

/** A focus with a title short enough that its pill is never at its widest. */
const shortFocus: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Ports', relation: 'focus' },
    { id: 'parent-0', title: 'Software architecture', relation: 'parent' },
    { id: 'child-0', title: 'Domain', relation: 'child' },
    { id: 'child-1', title: 'Adapter', relation: 'child' },
    { id: 'jump-0', title: 'Onion architecture', relation: 'jump' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0' },
    { from: 'focus', to: 'child-1' },
    { from: 'jump-0', to: 'focus' },
  ],
}

const getFocus = (frame: PlexFrame): PlacedNode => {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!focus) throw new Error('unreachable: every frame places a focus')
  return focus
}

const getNeighbors = (frame: PlexFrame): PlacedNode[] =>
  frame.nodes.filter((node) => node.relation !== 'focus')

const getBoxes = (nodes: readonly PlacedNode[]) =>
  nodes.map((node) => [node.id, node.width, node.height] as const)

function isOverlapping(a: PlacedNode, b: PlacedNode): boolean {
  return (
    Math.abs(a.x - b.x) < (a.width + b.width) / 2 - 1e-9 &&
    Math.abs(a.y - b.y) < (a.height + b.height) / 2 - 1e-9
  )
}

const countOverlaps = (frame: PlexFrame): number =>
  frame.nodes.reduce(
    (count, a, index) =>
      count + frame.nodes.slice(index + 1).filter((b) => isOverlapping(a, b)).length,
    0,
  )

const getChildSeparation = (frame: PlexFrame): number => {
  const children = frame.nodes.filter((node) => node.relation === 'child')
  let smallest = Infinity
  children.forEach((a, index) => {
    for (const b of children.slice(index + 1)) {
      const sideways = Math.abs(a.x - b.x) - (a.width + b.width) / 2
      const lengthways = Math.abs(a.y - b.y) - (a.height + b.height) / 2
      smallest = Math.min(smallest, Math.max(sideways, lengthways))
    }
  })
  return smallest
}

const arrange = (
  graph: PlexGraph,
  options: PlexOptionsInput = {},
  viewport: { width: number; height: number } | undefined = LARGE,
) => arrangePlex(graph, { options: { ...options, viewport }, measure: byTitle })

describe('the default look', () => {
  const frame = arrange(graphs.typical)
  const byId = new Map(frame.nodes.map((node) => [node.id, node]))

  it('draws every neighbor as wide as before and 0.05 taller', () => {
    for (const [id, width, height] of TODAY.slice(1)) {
      const node = byId.get(id)!
      expect(node.width, `${id} width`).toBe(width)
      expect(node.height, `${id} height`).toBeCloseTo(height + 0.05, 6)
    }
  })

  it('draws the focus pill 1.22 times as tall as a neighbor, and at most 1.22 times as wide as the widest neighbor', () => {
    const focus = getFocus(frame)
    const neighbor = byId.get('parent-0')!

    expect(focus.height).toBeCloseTo(neighbor.height * 1.22, 6)
    expect(focus.width).toBeCloseTo(144 * 1.22, 6)
  })

  it('draws a short focus title in a pill 1.22 times its measured width', () => {
    const focus = getFocus(arrange(shortFocus))

    expect(focus.width).toBeCloseTo(byTitle(shortFocus.nodes[0]!) * 1.22, 6)
  })

  it('joins the same nodes by its lines', () => {
    expect(frame.edges.map((edge) => [edge.from, edge.to])).toStrictEqual(TODAY_EDGES)
  })

  it('is the look the arrangement takes when no look is given', () => {
    const given = arrange(graphs.typical, {
      focusScale: 1.22,
      padding: { inline: 10, block: 9.25 },
      room: 0,
    })

    expect(given).toStrictEqual(frame)
  })
})

describe('Pill padding, top and bottom', () => {
  it('makes every box taller by twice the difference at the designed type', () => {
    const low = arrange(graphs.typical, { padding: { inline: 10, block: 2 } })
    const high = arrange(graphs.typical, { padding: { inline: 10, block: 12 } })
    const lowById = new Map(low.nodes.map((node) => [node.id, node]))

    for (const node of getNeighbors(high)) {
      expect(node.height - lowById.get(node.id)!.height, node.id).toBeCloseTo(20, 6)
    }
  })

  it('makes the focus taller by twice the difference times Focus size', () => {
    const options = { focusScale: 1.5 }
    const low = arrange(graphs.typical, { ...options, padding: { inline: 10, block: 2 } })
    const high = arrange(graphs.typical, { ...options, padding: { inline: 10, block: 12 } })

    expect(getFocus(high).height - getFocus(low).height).toBeCloseTo(30, 6)
  })
})

describe('Focus size', () => {
  const small = arrange(shortFocus, { focusScale: 1 })
  const large = arrange(shortFocus, { focusScale: 2 })

  it('draws the focus as tall as a neighbor at 1', () => {
    expect(getFocus(small).height).toBeCloseTo(getNeighbors(small)[0]!.height, 6)
  })

  it('draws the focus pill twice as tall and twice as wide at 2 as at 1', () => {
    expect(getFocus(large).height).toBeCloseTo(2 * getFocus(small).height, 6)
    expect(getFocus(large).width).toBeCloseTo(2 * getFocus(small).width, 6)
  })

  it('leaves every neighbor box as it was', () => {
    const sort = (boxes: ReturnType<typeof getBoxes>) =>
      [...boxes].sort(([a], [b]) => a.localeCompare(b))
    expect(sort(getBoxes(getNeighbors(large)))).toStrictEqual(sort(getBoxes(getNeighbors(small))))
  })

  it('keeps a large focus inside a small window', () => {
    const viewport = { width: 420, height: 300 }
    const focus = getFocus(arrange(graphs.typical, { focusScale: 2.5 }, viewport))

    expect(focus.width / 2).toBeLessThanOrEqual(viewport.width / 2)
    expect(focus.height / 2).toBeLessThanOrEqual(viewport.height / 2)
  })

  it('lets no box overlap another at the largest focus', () => {
    expect(countOverlaps(arrange(graphs.typical, { focusScale: 2.5 }))).toBe(0)
  })
})

describe('Room around a note', () => {
  const crowd = build('A note', { parent: 2, child: 8, jump: 3, sibling: 2 })

  it('moves neighboring boxes farther apart in a large window', () => {
    const viewport = { width: 4000, height: 3000 }
    const close = getChildSeparation(arrange(crowd, { room: 0 }, viewport))
    const sparse = getChildSeparation(arrange(crowd, { room: 20 }, viewport))

    expect(close).toBeLessThan(Infinity)
    expect(sparse).toBeGreaterThan(close)
  })

  it('keeps every box clear of every other at the widest room', () => {
    expect(countOverlaps(arrange(crowd, { room: 40 }, { width: 4000, height: 3000 }))).toBe(0)
  })

  it('moves the boxes apart where no window limits them', () => {
    const unlimited = (room: number) =>
      getChildSeparation(arrangePlex(crowd, { options: { room }, measure: byTitle }))
    const close = unlimited(0)
    const sparse = unlimited(20)

    // Free spots may sit half a unit inside a gap, so the difference is the room on both sides to within a unit.
    expect(Math.abs(sparse - close - 40)).toBeLessThanOrEqual(1)
  })

  it('shrinks with the gaps in a tight window, and still lets no box overlap', () => {
    const tight = { width: 700, height: 500 }
    const frame = arrange(build('A note', { child: 12 }), { room: 20 }, tight)
    const separation = getChildSeparation(frame)

    expect(separation).toBeLessThan(16 + 2 * 20)
    expect(countOverlaps(frame)).toBe(0)
  })

  it('counts the notes that still do not fit as not shown', () => {
    const tight = { width: 520, height: 360 }
    const frame = arrange(build('A note', { child: 60 }), { room: 40 }, tight)
    const drawn = frame.nodes.filter((node) => node.relation === 'child').length

    expect(frame.overflow.child).toBeGreaterThan(0)
    expect(drawn + frame.overflow.child!).toBe(60)
  })
})

describe('a look drawn at twice the text size', () => {
  const input: PlexOptionsInput = {
    focusScale: 1.5,
    padding: { inline: 4, block: 6 },
    room: 10,
    routing: { curvature: 0.3 },
  }
  const designed = optionsForType(DESIGNED_TYPE, resolveOptions(input))
  const doubled = optionsForType(2 * DESIGNED_TYPE, resolveOptions(input))

  it('doubles the padding and the room', () => {
    expect(doubled.padding.inline).toBeCloseTo(2 * designed.padding.inline, 6)
    expect(doubled.padding.block).toBeCloseTo(2 * designed.padding.block, 6)
    expect(doubled.room).toBeCloseTo(2 * designed.room, 6)
  })

  it('keeps Focus size and curvature as they are', () => {
    expect(doubled.focusScale).toBe(1.5)
    expect(doubled.routing.curvature).toBe(0.3)
  })

  it('doubles every box it draws', () => {
    const small = arrangePlex(graphs.typical, { options: designed })
    const large = arrangePlex(graphs.typical, { options: doubled })
    const smallById = new Map(small.nodes.map((node) => [node.id, node]))

    for (const node of large.nodes) {
      const was = smallById.get(node.id)!
      expect(node.width, `${node.id} width`).toBeCloseTo(2 * was.width, 6)
      expect(node.height, `${node.id} height`).toBeCloseTo(2 * was.height, 6)
    }
  })
})
