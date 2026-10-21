/** How a line is drawn between two boxes. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { cutToFit, routeEdge, routeEdges, createRouting } from './routing'
import { neighbourhoods } from '../../fixtures/neighbourhoods'
import { ARROW_LENGTH, getArrow, getLength, createPointOnCurve, type EdgeArrow } from '../edge'
import type { PlexNeighbourhood } from '../neighbourhood'
import type { PlacedNode } from '../node'
import type { PlexNodeRelation } from '../relation'

const getNodesByRelation = (frame: { nodes: readonly PlacedNode[] }, relation: PlexNodeRelation) =>
  frame.nodes.filter((node) => node.relation === relation)

const getFocus = (frame: { nodes: readonly PlacedNode[] }) => {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!focus) throw new Error('unreachable: every frame places a focus')
  return focus
}
describe('edges', () => {
  it('leaves the focus by one gate and arrives at the top of each child', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = getFocus(layout)
    const children = getNodesByRelation(layout, 'child')

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
    const outermost = getNodesByRelation(layout, 'child').reduce((a, b) =>
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

  it('runs sideways for a relation that runs sideways', () => {
    const layout = arrangePlex(neighbourhoods.typical)
    const focus = getFocus(layout)
    const jump = getNodesByRelation(layout, 'jump')[0]!
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
          (touchesSide || touchesTop) && dx <= node.width / 2 + 1e-9 && dy <= node.height / 2 + 1e-9
        )
      }
      expect(onBorder(edge.fromPoint, from), `from ${edge.from}`).toBe(true)
      expect(onBorder(edge.toPoint, to), `to ${edge.to}`).toBe(true)
    }
  })

  it('joins two nodes on the same line side to side, not under and back', () => {
    // The relation says vertical, but there is no vertical room between them.
    const layout = arrangePlex(neighbourhoods.diamond)
    const edge = layout.edges.find((e) => e.from === 'parent-0' && e.to === 'parent-1')
    expect(edge).toBeDefined()
    expect(edge!.label).toBe('contains')

    const left = layout.nodes.find((n) => n.id === 'parent-0')!
    const right = layout.nodes.find((n) => n.id === 'parent-1')!
    expect(edge!.fromPoint).toStrictEqual({ x: left.x + left.width / 2, y: left.y })
    expect(edge!.toPoint).toStrictEqual({ x: right.x - right.width / 2, y: right.y })
    expect(edge!.control1.x).toBeGreaterThan(edge!.fromPoint.x)
    expect(edge!.control2.x).toBeLessThan(edge!.toPoint.x)
  })

  it('takes the heading from the curve, and not from the relation', () => {
    const layout = arrangePlex(neighbourhoods.typical)

    // A sibling sits under a parent, and its line runs left to right.
    for (const sibling of getNodesByRelation(layout, 'sibling')) {
      const edge = layout.edges.find((e) => e.to === sibling.id)!
      expect(edge.heading, `sibling ${sibling.id}`).toBe('along')
    }

    // A jump sits sideways, and its line arrives from the left.
    for (const jump of getNodesByRelation(layout, 'jump')) {
      const edge = layout.edges.find((e) => e.from === jump.id)!
      expect(edge.heading, `jump ${jump.id}`).toBe('along')
    }
  })

  it('takes a curve running up the page the other way round', () => {
    // A parent is above the focus, and the line is written from it downwards.
    const layout = arrangePlex(neighbourhoods.typical)
    const down = layout.edges.find((e) => e.to === 'focus' && e.from === 'parent-0')!
    expect(down.fromPoint.y).toBeLessThan(down.toPoint.y)
    expect(down.heading).toBe('along')

    // The same pair written the other way about runs up it, and is read down.
    const up = arrangePlex({
      nodes: [
        { id: 'focus', title: 'Here', relation: 'focus' },
        { id: 'over', title: 'Over', relation: 'parent' },
      ],
      edges: [{ from: 'focus', to: 'over' }],
    })
    expect(up.edges[0]!.fromPoint.y).toBeGreaterThan(up.edges[0]!.toPoint.y)
    expect(up.edges[0]!.heading).toBe('against')
  })

  it('takes the reading direction from the curve, not from which end is which', () => {
    // One jump sits level with the focus, so the line between them is straight
    // and runs whichever way the pair was written.
    const nodes = [
      { id: 'focus', title: 'Here', relation: 'focus' as const },
      { id: 'aside', title: 'Aside', relation: 'jump' as const },
    ]
    const outward = arrangePlex({ nodes, edges: [{ from: 'focus', to: 'aside' }] })
    const inward = arrangePlex({ nodes, edges: [{ from: 'aside', to: 'focus' }] })

    expect(outward.edges[0]!.heading).toBe('against')
    expect(outward.edges[0]!.fromPoint.x).toBeGreaterThan(outward.edges[0]!.toPoint.x)
    expect(inward.edges[0]!.heading).toBe('along')
    expect(inward.edges[0]!.fromPoint.x).toBeLessThan(inward.edges[0]!.toPoint.x)
  })

  it('drops an edge from a node to itself', () => {
    const nodes = [{ id: 'a', title: 'A', relation: 'focus' as const }]
    const layout = arrangePlex({ nodes, edges: [{ from: 'a', to: 'a' }] })
    expect(layout.edges).toHaveLength(0)
  })
})

/**
 * A title is always set along its line, so words longer than the line are cut
 * to it. What the person wrote is kept; what is drawn is the cut.
 */
describe('a title cut to the line it is set on', () => {
  const sideways: PlexNeighbourhood = {
    nodes: [
      { id: 'focus', title: 'Here', relation: 'focus' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [{ from: 'aside', to: 'focus', label: 'the scene in the assembly' }],
  }

  const placed = arrangePlex(sideways)
  const byId = new Map(placed.nodes.map((node) => [node.id, node]))
  const curve = placed.edges[0]!
  const arc = getLength(curve)

  const routeWithLabel = (measureLabel?: (label: string) => number) =>
    routeEdges(sideways.edges, byId, createRouting(DEFAULT_OPTIONS, measureLabel))[0]!

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

  /** A letter of a fixed width, so what fits is a matter of counting. */
  const perLetter = (width: number) => (label: string) => width * [...label].length

  it('draws the whole label where it fits the curve', () => {
    const edge = routeWithLabel(perLetter(1))
    expect(edge.words).toBe('the scene in the assembly')
    expect(edge.label).toBe('the scene in the assembly')
  })

  it('draws the whole label where nothing measured it', () => {
    expect(routeWithLabel().words).toBe('the scene in the assembly')
  })

  it('cuts words longer than the curve, and ends them in an ellipsis', () => {
    const width = perLetter(arc / 10)
    const edge = routeWithLabel(width)

    expect(edge.words).not.toBe(edge.label)
    expect(edge.words!.endsWith('…')).toBe(true)
    expect(edge.label!.startsWith(edge.words!.slice(0, -1))).toBe(true)
    expect(width(edge.words!)).toBeLessThanOrEqual(arc)
  })

  it('leaves no space hanging before the ellipsis', () => {
    // Ten letters' room, and the tenth letter of this label is a space.
    const edge = routeWithLabel(perLetter(arc / 10))
    expect(edge.words).toBe('the scene…')
  })

  it('carries the ellipsis alone where there is room for nothing', () => {
    expect(routeWithLabel(perLetter(arc)).words).toBe('…')
  })

  it('measures the words it carries, and no other', () => {
    const asked: string[] = []
    routeWithLabel((label) => {
      asked.push(label)
      return 0
    })
    expect(asked).toStrictEqual(['the scene in the assembly'])
  })

  it('halves in on the cut and not stepping through the label', () => {
    const label = 'a'.repeat(400)
    const asked: string[] = []
    routeEdges(
      [{ from: 'aside', to: 'focus', label }],
      byId,
      createRouting(DEFAULT_OPTIONS, (words) => {
        asked.push(words)
        return [...words].length
      }),
    )
    expect(asked.length).toBeLessThan(20)
  })
})

/**
 * An arrowhead is drawn at one end of the line and aimed out of it. Which end
 * comes down with the edge; where and which way round is worked out here.
 */
describe('the arrowhead a line carries', () => {
  const nodes = [
    { id: 'focus', title: 'Here', relation: 'focus' as const },
    { id: 'below', title: 'Below', relation: 'child' as const },
  ]

  /** One line straight down the page, so its ends are a quarter turn apart. */
  const routeWithArrow = (arrow?: EdgeArrow) =>
    arrangePlex({
      nodes,
      edges: [{ from: 'focus', to: 'below', ...(arrow ? { arrow } : {}) }],
    }).edges[0]!

  it('sits where the line arrives, aimed the way it is going', () => {
    const edge = routeWithArrow('to')
    expect(edge.arrowhead!.at.x).toBe(edge.toPoint.x)
    expect(edge.arrowhead!.at.y).toBe(edge.toPoint.y)
    expect(edge.arrowhead!.angle).toBeCloseTo(90)
  })

  it('sits where the line leaves, aimed back out of it', () => {
    const edge = routeWithArrow('from')
    expect(edge.arrowhead!.at.x).toBe(edge.fromPoint.x)
    expect(edge.arrowhead!.at.y).toBe(edge.fromPoint.y)
    expect(edge.arrowhead!.angle).toBeCloseTo(-90)
  })

  it('is drawn on no line that was given no arrow', () => {
    expect(routeWithArrow().arrowhead).toBeUndefined()
  })

  // A head has a body, and over its length a curve turns, so a head aimed
  // along the piece of curve it covers has the line running into its base.
  it('is aimed along the piece of curve it covers', () => {
    const aside = arrangePlex({
      nodes: [
        { id: 'focus', title: 'Here', relation: 'focus' },
        { id: 'wide', title: 'Wide', relation: 'child' },
        { id: 'other', title: 'Other', relation: 'child' },
        { id: 'third', title: 'Third', relation: 'child' },
      ],
      edges: [
        { from: 'focus', to: 'wide', arrow: 'to' },
        { from: 'focus', to: 'other' },
        { from: 'focus', to: 'third' },
      ],
    }).edges.find((edge) => edge.to === 'wide')!

    const covered = ARROW_LENGTH / getLength(aside)
    const behindPoint = createPointOnCurve(aside)(1 - covered)
    const chord =
      (Math.atan2(aside.toPoint.y - behindPoint.y, aside.toPoint.x - behindPoint.x) * 180) / Math.PI

    // The line bends over the head's length, so the two readings differ.
    expect(Math.abs(chord - 90)).toBeGreaterThan(3)
    expect(aside.arrowhead!.angle).toBeCloseTo(chord, 1)

    // The end a line leaves from is read the same way, backwards.
    const leaving = getArrow(aside, 'from')
    const alongEdge = createPointOnCurve(aside)(ARROW_LENGTH / getLength(aside))
    const back =
      (Math.atan2(aside.fromPoint.y - alongEdge.y, aside.fromPoint.x - alongEdge.x) * 180) / Math.PI
    expect(leaving.angle).toBeCloseTo(back, 1)
  })

  // A curve shorter than the head it carries has no piece of itself to read, so
  // the tangent where the head points is what aims it.
  it('is aimed at the tangent where the line is shorter than the head', () => {
    const gate = { x: 0, y: 0 }
    const short = {
      fromPoint: gate,
      control1: { x: 0, y: 2 },
      control2: { x: 0, y: 4 },
      toPoint: { x: 0, y: 6 },
    }
    expect(getLength(short)).toBeLessThan(ARROW_LENGTH)
    expect(getArrow(short, 'to').angle).toBeCloseTo(90)
    expect(getArrow(short, 'from').angle).toBeCloseTo(-90)
  })

  it('leaves the title of its line short of the head', () => {
    const measure = (words: string) => 4 * [...words].length
    const byId = new Map(arrangePlex({ nodes, edges: [] }).nodes.map((node) => [node.id, node]))
    const words = (arrow?: EdgeArrow) =>
      routeEdges(
        [
          {
            from: 'focus',
            to: 'below',
            label: 'the scene in the assembly',
            ...(arrow ? { arrow } : {}),
          },
        ],
        byId,
        createRouting(DEFAULT_OPTIONS, measure),
      )[0]!.words!

    const room = getLength(routeWithArrow()) - 2 * DEFAULT_OPTIONS.routing.arrowRoom
    expect(measure(words('to'))).toBeLessThanOrEqual(room)
    expect(measure(words())).toBeGreaterThan(room)
  })
})

describe('the axis a relation is routed along', () => {
  const getAxis = createRouting(DEFAULT_OPTIONS).getAxis

  it('is left to the geometry for the focus', () => {
    expect(getAxis({ relation: 'focus' })).toBe('auto')
  })

  it('runs down the page for the relations that grow up or down', () => {
    expect(getAxis({ relation: 'parent' })).toBe('vertical')
    expect(getAxis({ relation: 'child' })).toBe('vertical')
  })

  it('runs across the page for the relations that grow left or right', () => {
    expect(getAxis({ relation: 'jump' })).toBe('horizontal')
    expect(getAxis({ relation: 'sibling' })).toBe('horizontal')
  })

  it('follows the direction a relation has been given', () => {
    const turned = createRouting({
      ...DEFAULT_OPTIONS,
      direction: { ...DEFAULT_OPTIONS.direction, child: 'left', jump: 'up' },
    })
    expect(turned.getAxis({ relation: 'child' })).toBe('horizontal')
    expect(turned.getAxis({ relation: 'jump' })).toBe('vertical')
  })
})

describe('cutting a title to a room', () => {
  const perLetter = (label: string) => [...label].length

  it('keeps a label that fills the room exactly', () => {
    expect(cutToFit('abc', 3, perLetter)).toBe('abc')
  })

  it('trims the end of the cut and leaves its start as written', () => {
    expect(cutToFit('  ab cd', 4, perLetter)).toBe('  a…')
  })
})

/**
 * Which way a line runs between two boxes is the axis of their relations, unless
 * the boxes leave no clear space along it and do along the other.
 */
describe('the run of a line between two boxes', () => {
  const box = (id: string, relation: PlexNodeRelation, x: number, y: number): PlacedNode => ({
    id,
    title: id,
    relation,
    x,
    y,
    width: 100,
    height: 40,
    order: 0,
    opacity: 1,
  })

  const routing = createRouting(DEFAULT_OPTIONS)

  const route = (from: PlacedNode, to: PlacedNode, using = routing) =>
    routeEdge(
      { from: from.id, to: to.id },
      new Map([from, to].map((node) => [node.id, node])),
      using,
    )!

  it('runs down the page for a vertical relation with clear space below', () => {
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 300, 60))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 300, y: 40 })
  })

  it('runs across the page where boxes of a vertical relation only touch', () => {
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 300, 40))

    expect(edge.fromPoint).toStrictEqual({ x: 50, y: 0 })
    expect(edge.toPoint).toStrictEqual({ x: 250, y: 40 })
  })

  it('runs down the page where boxes overlap in both directions', () => {
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 30, 10))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 30, y: -10 })
  })

  it('runs across the page for a horizontal relation with clear space alongside it', () => {
    const edge = route(box('a', 'jump', 0, 0), box('b', 'jump', 150, 200))

    expect(edge.fromPoint).toStrictEqual({ x: 50, y: 0 })
    expect(edge.toPoint).toStrictEqual({ x: 100, y: 200 })
  })

  it('runs down the page where boxes of a horizontal relation only touch', () => {
    const edge = route(box('a', 'jump', 0, 0), box('b', 'jump', 100, 200))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 100, y: 180 })
  })

  it('takes the axis of the relation that has one, whichever end it is', () => {
    // Further across than down, and still a line to a child hangs from below.
    const edge = route(box('focus', 'focus', 0, 0), box('child', 'child', 500, 100))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 500, y: 80 })
  })

  it('takes the axis from the geometry where neither relation has one', () => {
    const undeclared = { ...routing, getAxis: () => 'auto' as const }

    const down = route(box('a', 'focus', 0, 0), box('b', 'focus', 150, 200), undeclared)
    expect(down.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(down.toPoint).toStrictEqual({ x: 150, y: 180 })

    const acrossRoute = route(box('a', 'focus', 0, 0), box('b', 'focus', 300, 60), undeclared)
    expect(acrossRoute.fromPoint).toStrictEqual({ x: 50, y: 0 })
    expect(acrossRoute.toPoint).toStrictEqual({ x: 250, y: 60 })

    // Equally far across and down, the run is down the page.
    const level = route(box('a', 'focus', 0, 0), box('b', 'focus', 150, 150), undeclared)
    expect(level.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(level.toPoint).toStrictEqual({ x: 150, y: 130 })

    // The distances are those between the boxes, wherever they stand.
    const farDown = route(box('a', 'focus', 0, 300), box('b', 'focus', 200, 400), undeclared)
    expect(farDown.fromPoint).toStrictEqual({ x: 50, y: 300 })
    expect(farDown.toPoint).toStrictEqual({ x: 150, y: 400 })

    const farAcross = route(box('a', 'focus', 300, 0), box('b', 'focus', 420, 200), undeclared)
    expect(farAcross.fromPoint).toStrictEqual({ x: 300, y: 20 })
    expect(farAcross.toPoint).toStrictEqual({ x: 420, y: 180 })
  })

  it('leaves from the top of a box that is below the one it arrives at', () => {
    const edge = route(box('low', 'child', 0, 200), box('high', 'child', 0, 0))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 180 })
    expect(edge.toPoint).toStrictEqual({ x: 0, y: 20 })
  })

  it('leaves from the bottom of a box level with the one it arrives at', () => {
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 10, 0))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 10, y: -20 })
  })

  it('leaves from the right of a box at the same height alongside the one it arrives at', () => {
    const edge = route(box('a', 'jump', 0, 0), box('b', 'jump', 0, 10))

    expect(edge.fromPoint).toStrictEqual({ x: 50, y: 0 })
    expect(edge.toPoint).toStrictEqual({ x: -50, y: 10 })
  })

  it('holds the leaving direction for a fraction of the gap between the gates', () => {
    const half = { ...routing, curvature: 0.5, minReach: 0 }
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 0, 200), half)

    // The gates are 160 apart, so each control stands 80 out of its gate.
    expect(edge.control1).toStrictEqual({ x: 0, y: 100 })
    expect(edge.control2).toStrictEqual({ x: 0, y: 100 })
  })

  it('draws a line fully opaque unless it is given an opacity', () => {
    const a = box('a', 'child', 0, 0)
    const b = box('b', 'child', 0, 200)
    const byId = new Map([a, b].map((node) => [node.id, node]))

    expect(routeEdges([{ from: 'a', to: 'b' }], byId, routing)[0]!.opacity).toBe(1)
    expect(
      routeEdges([{ from: 'a', to: 'b' }], byId, { routing, opacity: () => 0.25 })[0]!.opacity,
    ).toBe(0.25)
  })
})
