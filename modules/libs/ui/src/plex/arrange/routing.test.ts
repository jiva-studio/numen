/** How a line is drawn between two boxes. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { routeEdges, routingFor } from './routing'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import {
  lengthOf,
  type PlacedNode,
  type PlexNeighbourhood,
  type PlexSeat,
} from '../model'

const withSeat = (frame: { nodes: readonly PlacedNode[] }, seat: PlexSeat) =>
  frame.nodes.filter((node) => node.seat === seat)

const focusOf = (frame: { nodes: readonly PlacedNode[] }) => {
  const focus = frame.nodes.find((node) => node.seat === 'focus')
  if (!focus) throw new Error('unreachable: every frame places a focus')
  return focus
}
describe('edges', () => {
  it('leaves the focus by one gate and arrives at the top of each child', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = focusOf(layout)
    const children = withSeat(layout, 'child')

    for (const child of children) {
      const edge = layout.edges.find((e) => e.to === child.id)
      expect(edge, `no edge to ${child.id}`).toBeDefined()

      // Every child hangs from the same point under the focus...
      expect(edge!.fromPoint).toStrictEqual({ x: focus.x, y: focus.y + focus.height / 2 })
      // ...and meets its own box square on top, however far sideways it sits.
      expect(edge!.toPoint).toStrictEqual({ x: child.x, y: child.y - child.height / 2 })
    }
  })

  it('sets off and arrives along the axis, never across it', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const outermost = withSeat(layout, 'child').reduce((a, b) =>
      Math.abs(a.x) > Math.abs(b.x) ? a : b,
    )
    const edge = layout.edges.find((e) => e.to === outermost.id)!

    // Reading the axis off the distance flips the outermost child, which is
    // further sideways than it is down.
    expect(Math.abs(outermost.x)).toBeGreaterThan(Math.abs(outermost.y))
    expect(edge.control1.x).toBe(edge.fromPoint.x)
    expect(edge.control1.y).toBeGreaterThan(edge.fromPoint.y)
    expect(edge.control2.x).toBe(edge.toPoint.x)
    expect(edge.control2.y).toBeLessThan(edge.toPoint.y)
  })

  it('runs sideways for a seat that is seated sideways', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = focusOf(layout)
    const jump = withSeat(layout, 'jump')[0]!
    const edge = layout.edges.find((e) => e.from === jump.id)!

    expect(edge.fromPoint).toStrictEqual({ x: jump.x + jump.width / 2, y: jump.y })
    expect(edge.toPoint).toStrictEqual({ x: focus.x - focus.width / 2, y: focus.y })
    expect(edge.control1.y).toBe(edge.fromPoint.y)
    expect(edge.control2.y).toBe(edge.toPoint.y)
  })

  it('starts and ends on a border, not in a centre', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const byId = new Map(layout.nodes.map((node) => [node.id, node]))

    for (const edge of layout.edges) {
      const from = byId.get(edge.from)!
      const to = byId.get(edge.to)!
      // On the border means: on one pair of sides, and within the other.
      const onBorder = (point: { x: number; y: number }, node: PlacedNode) => {
        const dx = Math.abs(point.x - node.x)
        const dy = Math.abs(point.y - node.y)
        const touchesSide = Math.abs(dx - node.width / 2) < 1e-9
        const touchesTop = Math.abs(dy - node.height / 2) < 1e-9
        return (
          (touchesSide || touchesTop) &&
          dx <= node.width / 2 + 1e-9 &&
          dy <= node.height / 2 + 1e-9
        )
      }
      expect(onBorder(edge.fromPoint, from), `from ${edge.from}`).toBe(true)
      expect(onBorder(edge.toPoint, to), `to ${edge.to}`).toBe(true)
    }
  })

  it('joins two nodes on the same line side to side, not under and back', () => {
    // The seat says vertical, but there is no vertical room between them.
    const layout = arrangePlex(neighbourhoods.diamond)
    const edge = layout.edges.find(
      (e) => e.from === 'parent-0' && e.to === 'parent-1',
    )
    expect(edge).toBeDefined()
    expect(edge!.label).toBe('contains')

    const left = layout.nodes.find((n) => n.id === 'parent-0')!
    const right = layout.nodes.find((n) => n.id === 'parent-1')!
    expect(edge!.fromPoint).toStrictEqual({ x: left.x + left.width / 2, y: left.y })
    expect(edge!.toPoint).toStrictEqual({ x: right.x - right.width / 2, y: right.y })
    expect(edge!.control1.x).toBeGreaterThan(edge!.fromPoint.x)
    expect(edge!.control2.x).toBeLessThan(edge!.toPoint.x)
  })

  it('takes the heading from the curve, and not from the seat', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = focusOf(layout)

    // A jump is seated sideways. The ones above and below the focus leave
    // their column and come round, and at the midpoint they run down the page.
    const offRow = withSeat(layout, 'jump').filter((node) => node.y !== focus.y)
    expect(offRow.length).toBeGreaterThan(0)
    for (const jump of offRow) {
      const edge = layout.edges.find((e) => e.from === jump.id)!
      expect(edge.heading, `jump ${jump.id}`).toBe('none')
    }

    // A sibling is seated under a parent, and runs across the page.
    for (const sibling of withSeat(layout, 'sibling')) {
      const edge = layout.edges.find((e) => e.to === sibling.id)!
      expect(edge.heading, `sibling ${sibling.id}`).toBe('right')
    }
  })

  it('takes no heading from a curve that doubles back on itself', () => {
    // Two boxes side by side in one row sit closer together than the reach an
    // edge leaves with, so their gates are passed before the curve turns for
    // them. The one word this edge carries is set flat.
    const layout = arrangePlex(neighbourhoods.diamond)
    const doubled = layout.edges.find(
      (e) => e.from === 'parent-0' && e.to === 'parent-1',
    )!
    expect(doubled.label).toBe('contains')
    expect(doubled.heading).toBe('none')

    // The steepest curve in the same picture that still reads is left alone.
    const leaning = layout.edges.find((e) => e.from === 'parent-1' && e.to === 'focus')!
    expect(leaning.heading).toBe('left')
  })

  it('takes the reading direction from the curve, not from which end is which', () => {
    // One jump sits level with the focus, so the line between them is straight
    // and runs whichever way the pair was written.
    const nodes = [
      { id: 'focus', title: 'Here', seat: 'focus' as const },
      { id: 'aside', title: 'Aside', seat: 'jump' as const },
    ]
    const outward = arrangePlex({ nodes, edges: [{ from: 'focus', to: 'aside' }] })
    const inward = arrangePlex({ nodes, edges: [{ from: 'aside', to: 'focus' }] })

    expect(outward.edges[0]!.heading).toBe('left')
    expect(outward.edges[0]!.fromPoint.x).toBeGreaterThan(outward.edges[0]!.toPoint.x)
    expect(inward.edges[0]!.heading).toBe('right')
    expect(inward.edges[0]!.fromPoint.x).toBeLessThan(inward.edges[0]!.toPoint.x)
  })

  it('drops an edge from a node to itself', () => {
    const nodes = [{ id: 'a', title: 'A', seat: 'focus' as const }]
    const layout = arrangePlex({ nodes, edges: [{ from: 'a', to: 'a' }] })
    expect(layout.edges).toHaveLength(0)
  })
})

/**
 * A title set along a path keeps the glyphs that fall on it and drops the rest,
 * so a long one on a short line arrives as the middle of itself.
 */
describe('a title measured against the line it would be set on', () => {
  const sideways: PlexNeighbourhood = {
    nodes: [
      { id: 'focus', title: 'Here', seat: 'focus' },
      { id: 'aside', title: 'Aside', seat: 'jump' },
    ],
    edges: [{ from: 'aside', to: 'focus', label: 'the scene in the assembly' }],
  }

  const placed = arrangePlex(sideways)
  const byId = new Map(placed.nodes.map((node) => [node.id, node]))
  const curve = placed.edges[0]!
  const arc = lengthOf(curve)

  const routed = (measureLabel?: (label: string) => number) =>
    routeEdges(sideways.edges, byId, routingFor(DEFAULT_OPTIONS, measureLabel))[0]!

  it('estimates the curve between its chord and the way round its controls', () => {
    const step = (a: { x: number; y: number }, b: { x: number; y: number }) =>
      Math.hypot(b.x - a.x, b.y - a.y)
    const chord = step(curve.fromPoint, curve.toPoint)
    const round =
      step(curve.fromPoint, curve.control1) +
      step(curve.control1, curve.control2) +
      step(curve.control2, curve.toPoint)

    expect(arc).toBeGreaterThanOrEqual(chord)
    expect(arc).toBeLessThanOrEqual(round)
  })

  it('lies flat where the words are longer than the curve', () => {
    expect(routed(() => arc + 1).heading).toBe('none')
  })

  it('is set along the line where they fit on it', () => {
    expect(routed(() => arc - 1).heading).toBe('right')
  })

  it('is judged by direction and turn alone where there is no measurer', () => {
    expect(routed().heading).toBe('right')
  })

  it('measures the words it carries, and no other', () => {
    const asked: string[] = []
    routed((label) => {
      asked.push(label)
      return 0
    })
    expect(asked).toStrictEqual(['the scene in the assembly'])
  })
})

