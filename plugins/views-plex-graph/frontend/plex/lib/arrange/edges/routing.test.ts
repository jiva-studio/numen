/** How a line is drawn between two boxes. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { DEFAULT_OPTIONS, resolveOptions } from '../options'
import {
  cutToFit,
  getLabelPositions,
  getLabelPlaces,
  getRise,
  getShift,
  LABEL_GAP,
  MIDDLE,
  routeEdge,
  routeEdges,
  createRouting,
} from './routing'
import { neighborhoods } from '../../../fixtures/neighborhoods'
import { ARROW_LENGTH, getArrow, getHeading, getLength, createPointOnCurve } from '../../edge'
import type { PlacedNode } from '../../node'
import type { EdgeArrow, PlexEdge, PlexNeighborhood, PlexNodeRelation } from '../../../../types'

/** The ink of the label font, as the window measures it. */
const TYPE = { ascent: 9, descent: 3, xHeight: 6 }

const getNodesByRelation = (frame: { nodes: readonly PlacedNode[] }, relation: PlexNodeRelation) =>
  frame.nodes.filter((node) => node.relation === relation)

const getFocus = (frame: { nodes: readonly PlacedNode[] }) => {
  const focus = frame.nodes.find((node) => node.relation === 'focus')
  if (!focus) throw new Error('unreachable: every frame places a focus')
  return focus
}
describe('edges', () => {
  it('leaves the focus by one gate and arrives at the top of each child', () => {
    const layout = arrangePlex(neighborhoods.typical)
    const focus = getFocus(layout)
    const children = getNodesByRelation(layout, 'child')

    for (const child of children) {
      const edge = layout.edges.find((e) => e.to === child.id)
      expect(edge, `no edge to ${child.id}`).toBeDefined()

      // Every child hangs from the same point under the focus...
      expect(edge!.fromPoint).toStrictEqual({ x: focus.x, y: focus.y + focus.height / 2 })
      // ...and ends at the top of its own box, however far sideways it is offset.
      expect(edge!.toPoint).toStrictEqual({ x: child.x, y: child.y - child.height / 2 })
    }
  })

  it('sets off and arrives along the axis, never across it', () => {
    const layout = arrangePlex(neighborhoods.typical)
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
    const layout = arrangePlex(neighborhoods.typical)
    const focus = getFocus(layout)
    const jump = getNodesByRelation(layout, 'jump')[0]!
    const edge = layout.edges.find((e) => e.from === jump.id)!

    expect(edge.fromPoint).toStrictEqual({ x: jump.x + jump.width / 2, y: jump.y })
    expect(edge.toPoint).toStrictEqual({ x: focus.x - focus.width / 2, y: focus.y })
    expect(edge.control1.y).toBe(edge.fromPoint.y)
    expect(edge.control2.y).toBe(edge.toPoint.y)
  })

  it('starts and ends on a border, not in a center', () => {
    const layout = arrangePlex(neighborhoods.typical)
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

  it('joins two nodes on the same line with no room between them by a bow, not through the boxes', () => {
    // The relation is vertical, there is no vertical space between them, and the
    // gap beside them is too small for a line to carry its words and heads.
    const layout = arrangePlex(neighborhoods.diamond)
    const edge = layout.edges.find((e) => e.from === 'parent-0' && e.to === 'parent-1')
    expect(edge).toBeDefined()
    expect(edge!.labels?.[0]?.text).toBe('contains')

    const leftNode = layout.nodes.find((n) => n.id === 'parent-0')!
    const right = layout.nodes.find((n) => n.id === 'parent-1')!
    const bottom = (node: PlacedNode) => ({ x: node.x, y: node.y + node.height / 2 })
    expect(edge!.fromPoint).toStrictEqual(bottom(leftNode))
    expect(edge!.toPoint).toStrictEqual(bottom(right))
    expect(edge!.control1.y).toBeGreaterThan(edge!.fromPoint.y)
    expect(edge!.control2.y).toBeGreaterThan(edge!.toPoint.y)
  })

  it('takes the heading from the curve, and not from the relation', () => {
    const layout = arrangePlex(neighborhoods.typical)

    // A sibling is below a parent, and its line runs left to right.
    for (const sibling of getNodesByRelation(layout, 'sibling')) {
      const edge = layout.edges.find((e) => e.to === sibling.id)!
      expect(getHeading(edge, MIDDLE), `sibling ${sibling.id}`).toBe('along')
    }

    // A jump is offset sideways, and its line arrives from the left.
    for (const jump of getNodesByRelation(layout, 'jump')) {
      const edge = layout.edges.find((e) => e.from === jump.id)!
      expect(getHeading(edge, MIDDLE), `jump ${jump.id}`).toBe('along')
    }
  })

  it('takes a curve running up the page the other way round', () => {
    // A parent is above the focus, and the line is written from it downwards.
    const layout = arrangePlex(neighborhoods.typical)
    const down = layout.edges.find((e) => e.to === 'focus' && e.from === 'parent-0')!
    expect(down.fromPoint.y).toBeLessThan(down.toPoint.y)
    expect(getHeading(down, MIDDLE)).toBe('along')

    // The same pair written the other way about runs up it, and is read down.
    const up = arrangePlex({
      nodes: [
        { id: 'focus', title: 'Here', relation: 'focus' },
        { id: 'over', title: 'Over', relation: 'parent' },
      ],
      edges: [{ from: 'focus', to: 'over' }],
    })
    expect(up.edges[0]!.fromPoint.y).toBeGreaterThan(up.edges[0]!.toPoint.y)
    expect(getHeading(up.edges[0]!, MIDDLE)).toBe('against')
  })

  it('takes the reading direction from the curve, not from which end is which', () => {
    // One jump is level with the focus, so the line between them is straight and runs
    // the same way whichever order the pair was written.
    const nodes = [
      { id: 'focus', title: 'Here', relation: 'focus' as const },
      { id: 'aside', title: 'Aside', relation: 'jump' as const },
    ]
    const outward = arrangePlex({ nodes, edges: [{ from: 'focus', to: 'aside' }] })
    const inward = arrangePlex({ nodes, edges: [{ from: 'aside', to: 'focus' }] })

    expect(getHeading(outward.edges[0]!, MIDDLE)).toBe('against')
    expect(outward.edges[0]!.fromPoint.x).toBeGreaterThan(outward.edges[0]!.toPoint.x)
    expect(getHeading(inward.edges[0]!, MIDDLE)).toBe('along')
    expect(inward.edges[0]!.fromPoint.x).toBeLessThan(inward.edges[0]!.toPoint.x)
  })

  it('drops an edge from a node to itself', () => {
    const nodes = [{ id: 'a', title: 'A', relation: 'focus' as const }]
    const layout = arrangePlex({ nodes, edges: [{ from: 'a', to: 'a' }] })
    expect(layout.edges).toHaveLength(0)
  })
})

/**
 * A title is always set along its line, so words longer than the line are truncated
 * to it. The stored text is unchanged; the rendered text is the truncation.
 */
describe('a title cut to the line it is set on', () => {
  const sideways: PlexNeighborhood = {
    nodes: [
      { id: 'focus', title: 'Here', relation: 'focus' },
      { id: 'aside', title: 'Aside', relation: 'jump' },
    ],
    edges: [{ from: 'aside', to: 'focus', labels: [{ text: 'the scene in the assembly' }] }],
  }

  const placedPlex = arrangePlex(sideways)
  const byId = new Map(placedPlex.nodes.map((node) => [node.id, node]))
  const curve = placedPlex.edges[0]!
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
    expect(edge.titles[0]?.words).toBe('the scene in the assembly')
    expect(edge.labels?.[0]?.text).toBe('the scene in the assembly')
  })

  it('draws the whole label where nothing measured it', () => {
    expect(routeWithLabel().titles[0]?.words).toBe('the scene in the assembly')
  })

  it('cuts words longer than the curve, and ends them in an ellipsis', () => {
    const width = perLetter(arc / 10)
    const edge = routeWithLabel(width)

    expect(edge.titles[0]?.words).not.toBe(edge.labels?.[0]?.text)
    expect(edge.titles[0]!.words.endsWith('…')).toBe(true)
    expect(edge.labels![0]!.text.startsWith(edge.titles[0]!.words.slice(0, -1))).toBe(true)
    expect(width(edge.titles[0]!.words)).toBeLessThanOrEqual(arc)
  })

  it('leaves no space hanging before the ellipsis', () => {
    // Space for ten letters; the tenth letter of this label is a space.
    const edge = routeWithLabel(perLetter(arc / 10))
    expect(edge.titles[0]?.words).toBe('the scene…')
  })

  it('carries the ellipsis alone where there is room for nothing', () => {
    expect(routeWithLabel(perLetter(arc)).titles[0]?.words).toBe('…')
  })

  it('measures the words it carries, and no other', () => {
    const request: string[] = []
    routeWithLabel((label) => {
      request.push(label)
      return 0
    })
    expect(request).toStrictEqual(['the scene in the assembly'])
  })

  it('halves in on the cut and not stepping through the label', () => {
    const label = 'a'.repeat(400)
    const request: string[] = []
    routeEdges(
      [{ from: 'aside', to: 'focus', labels: [{ text: label }] }],
      byId,
      createRouting(DEFAULT_OPTIONS, (words) => {
        request.push(words)
        return [...words].length
      }),
    )
    expect(request.length).toBeLessThan(20)
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
      edges: [{ from: 'focus', to: 'below', ...(arrow ? { arrows: [arrow] } : {}) }],
    }).edges[0]!

  it('sits where the line arrives, aimed the way it is going', () => {
    const edge = routeWithArrow('to')
    expect(edge.arrowheads![0]!.at.x).toBe(edge.toPoint.x)
    expect(edge.arrowheads![0]!.at.y).toBe(edge.toPoint.y)
    expect(edge.arrowheads![0]!.angle).toBeCloseTo(90)
  })

  it('sits where the line leaves, aimed back out of it', () => {
    const edge = routeWithArrow('from')
    expect(edge.arrowheads![0]!.at.x).toBe(edge.fromPoint.x)
    expect(edge.arrowheads![0]!.at.y).toBe(edge.fromPoint.y)
    expect(edge.arrowheads![0]!.angle).toBeCloseTo(-90)
  })

  it('is drawn at both ends of a line given both', () => {
    const edge = arrangePlex({
      nodes,
      edges: [{ from: 'focus', to: 'below', arrows: ['from', 'to'] }],
    }).edges[0]!
    expect(edge.arrowheads!.map((head) => head.end)).toEqual(['from', 'to'])
    expect(edge.arrowheads![0]!.at.y).toBe(edge.fromPoint.y)
    expect(edge.arrowheads![1]!.at.y).toBe(edge.toPoint.y)
  })

  it('is drawn on no line that was given no arrow', () => {
    expect(routeWithArrow().arrowheads).toBeUndefined()
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
        { from: 'focus', to: 'wide' },
        { from: 'focus', to: 'other' },
        { from: 'focus', to: 'third', arrows: ['to'] },
      ],
    }).edges.find((edge) => edge.to === 'third')!

    const covered = ARROW_LENGTH / getLength(aside)
    const precedingPoint = createPointOnCurve(aside)(1 - covered)
    const chord =
      (Math.atan2(aside.toPoint.y - precedingPoint.y, aside.toPoint.x - precedingPoint.x) * 180) /
      Math.PI

    expect(Math.abs(chord - 90)).toBeGreaterThan(3)
    expect(aside.arrowheads![0]!.angle).toBeCloseTo(chord, 1)

    // The end a line leaves from is read the same way, backwards.
    const leaving = getArrow(aside, 'from')
    const axisEdge = createPointOnCurve(aside)(ARROW_LENGTH / getLength(aside))
    const back =
      (Math.atan2(aside.fromPoint.y - axisEdge.y, aside.fromPoint.x - axisEdge.x) * 180) / Math.PI
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
            labels: [{ text: 'the scene in the assembly' }],
            ...(arrow ? { arrows: [arrow] } : {}),
          },
        ],
        byId,
        createRouting(DEFAULT_OPTIONS, measure),
      )[0]!.titles[0]!.words

    const room = getLength(routeWithArrow()) - 2 * DEFAULT_OPTIONS.routing.arrowClearance
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

  const edgeRouting = createRouting(DEFAULT_OPTIONS)

  const route = (from: PlacedNode, to: PlacedNode, using = edgeRouting) =>
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
    const undeclared = { ...edgeRouting, getAxis: () => 'auto' as const }

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
    const half = {
      ...edgeRouting,
      tuning: { ...edgeRouting.tuning, curvature: 0.5, minReach: 0 },
    }
    const edge = route(box('a', 'child', 0, 0), box('b', 'child', 0, 200), half)

    // The gates are 160 apart, so each control is offset 80 from its gate.
    expect(edge.control1).toStrictEqual({ x: 0, y: 100 })
    expect(edge.control2).toStrictEqual({ x: 0, y: 100 })
  })

  it('draws a line fully opaque unless it is given an opacity', () => {
    const a = box('a', 'child', 0, 0)
    const b = box('b', 'child', 0, 200)
    const byId = new Map([a, b].map((node) => [node.id, node]))

    expect(routeEdges([{ from: 'a', to: 'b' }], byId, edgeRouting)[0]!.opacity).toBe(1)
    expect(
      routeEdges([{ from: 'a', to: 'b' }], byId, { routing: edgeRouting, opacity: () => 0.25 })[0]!
        .opacity,
    ).toBe(0.25)
  })
})

/**
 * The position of a label along its line is decided by how many labels the line
 * carries: one is centered, two are set at opposite ends. The arrowheads do not
 * enter into it.
 */
describe('the position of a label', () => {
  const box = (id: string, x: number): PlacedNode => ({
    id,
    title: id,
    relation: 'jump',
    x,
    y: 0,
    width: 100,
    height: 20,
    order: 0,
    opacity: 1,
  })
  const harbour = box('harbour', 0)
  const lighthouse = box('lighthouse', 400)
  const byId = new Map([harbour, lighthouse].map((node) => [node.id, node]))

  /** The titles of one line: words, and the place from the `from` end. */
  const getPlaces = (edge: PlexEdge) => {
    const routed = routeEdge(edge, byId, createRouting(DEFAULT_OPTIONS))!
    return routed.titles.map((title) => [
      title.words,
      title.heading === 'against' ? 1 - title.wordsAt : title.wordsAt,
    ])
  }

  it('is the middle of the line for one label', () => {
    expect(getLabelPositions(1)).toEqual([MIDDLE])
    expect(getPlaces({ from: 'harbour', to: 'lighthouse', labels: [{ text: 'guides' }] })).toEqual([
      ['guides', MIDDLE],
    ])
  })

  it('is the middle for one label whatever arrowhead the line has', () => {
    for (const arrows of [['to'], ['from'], ['to', 'from']] as const) {
      const edge: PlexEdge = {
        from: 'harbour',
        to: 'lighthouse',
        labels: [{ text: 'guides' }],
        arrows,
      }
      expect(getPlaces(edge)).toEqual([['guides', MIDDLE]])
    }
  })

  it('is one near each end for two labels, the first near the start', () => {
    const [start, end] = getLabelPositions(2)
    expect(start).toBeLessThan(1 / 3)
    expect(end).toBeGreaterThan(2 / 3)
    expect(start! + end!).toBeCloseTo(1)

    const edge: PlexEdge = {
      from: 'harbour',
      to: 'lighthouse',
      labels: [{ text: 'guides' }, { text: 'shelters' }],
      arrows: ['to', 'from'],
    }
    expect(getPlaces(edge)).toEqual([
      ['guides', start],
      ['shelters', end],
    ])
  })

  it('is the same for two labels whatever arrowheads the line has', () => {
    const labels = [{ text: 'guides' }, { text: 'shelters' }]
    const expected = getPlaces({ from: 'harbour', to: 'lighthouse', labels })

    for (const arrows of [['to'], ['from'], ['from', 'to']] as const) {
      expect(getPlaces({ from: 'harbour', to: 'lighthouse', labels, arrows })).toEqual(expected)
    }
  })

  it('keeps the first label near the `from` note on a line that runs right to left', () => {
    const edge: PlexEdge = {
      from: 'lighthouse',
      to: 'harbour',
      labels: [{ text: 'guides' }, { text: 'shelters' }],
    }
    const routed = routeEdge(edge, byId, createRouting(DEFAULT_OPTIONS))!
    const [start, end] = getLabelPositions(2)

    // The lighthouse is the `from` note, so the label it wrote is near the lighthouse.
    expect(routed.titles.map((title) => title.words)).toEqual(['guides', 'shelters'])
    // The lighthouse record reads from the lighthouse to the harbour: along the curve, which runs that way.
    expect(routed.titles[0]!.heading).toBe('along')
    expect(routed.titles[0]!.wordsAt).toBeCloseTo(start!)
    const places = getPlaces(edge)
    expect(places.map(([words]) => words)).toEqual(['guides', 'shelters'])
    expect(places[0]![1]).toBeCloseTo(start!)
    expect(places[1]![1]).toBeCloseTo(end!)
  })

  it('sets the two labels apart, at the two ends, on every kind of line', () => {
    const measureLabel = (words: string) => 6 * [...words].length
    let measured = 0

    for (const [name, neighborhood] of Object.entries(neighborhoods)) {
      const frame = arrangePlex(
        {
          ...neighborhood,
          edges: neighborhood.edges.map((edge) => ({
            ...edge,
            labels: [{ text: 'guides' }, { text: 'shelters' }],
          })),
        },
        { measureLabel, labelType: TYPE },
      )
      for (const edge of frame.edges) {
        const [one, other] = edge.titles
        if (!one || !other) continue
        measured += 1
        const place = (title: typeof one) =>
          title.heading === 'against' ? 1 - title.wordsAt : title.wordsAt
        const gap = Math.abs(place(one) - place(other)) * getLength(edge)

        expect(gap, `${name}: ${edge.from} to ${edge.to}`).toBeGreaterThanOrEqual(
          (measureLabel(one.words) + measureLabel(other.words)) / 2 - 1,
        )
      }
    }
    expect(measured).toBeGreaterThan(10)
  })

  it('sets one label on the line and two on opposite sides of it', () => {
    expect(getLabelPlaces([{ text: 'a' }]).map((place) => place.side)).toEqual([0])
    expect(getLabelPlaces([{ text: 'a' }, { text: 'b' }]).map((place) => place.side)).toEqual([
      -1, 1,
    ])
    expect(getShift(0, TYPE)).toBe(0)
    // Half the letters and the gap, to the side each label is on.
    expect(getShift(-1, TYPE)).toBe(-(6 + LABEL_GAP))
    expect(getShift(1, TYPE)).toBe(-getShift(-1, TYPE))
  })

  it('sets the baseline of the one label so the middle of its lowercase letters is on the line', () => {
    // The baseline is half the height of an x below the line, whatever else the font does.
    expect(getRise(0, TYPE)).toBe(3)
    expect(getRise(0, { ascent: 20, descent: 1, xHeight: 8 })).toBe(4)
    // A label beside the line is centered by the middle of its ink, from capital to descender.
    expect(getRise(-1, TYPE)).toBe((9 - 3) / 2)
    expect(getRise(1, { ascent: 10, descent: 4, xHeight: 6 })).toBe(3)
    // So the near edge of the ink is the gap from the line, on each side.
    expect(getShift(-1, TYPE) + getRise(-1, TYPE) + TYPE.descent).toBe(-LABEL_GAP)
    expect(getShift(1, TYPE) + getRise(1, TYPE) - TYPE.ascent).toBe(LABEL_GAP)
  })

  it('reads each of two labels in the direction of its record, whichever way the line runs', () => {
    const labels = [{ text: 'guides' }, { text: 'shelters' }]
    const measured = createRouting(DEFAULT_OPTIONS, undefined, TYPE)

    for (const [from, to] of [
      ['harbour', 'lighthouse'],
      ['lighthouse', 'harbour'],
    ] as const) {
      const [first, second] = routeEdge({ from, to, labels }, byId, measured)!.titles
      // The first record reads from `from` to `to`, on its left; the second the other way, on the
      // right of the line, which is the left of its own reading.
      expect([first!.heading, first!.side]).toEqual(['along', -1])
      expect([second!.heading, second!.side]).toEqual(['against', 1])
      expect(first!.shift).toBe(getShift(-1, TYPE))
      expect(second!.shift).toBe(getShift(1, TYPE))
    }
  })

  it('sets each of two labels near the end its head is at and reads it toward that end', () => {
    const [start, end] = getLabelPositions(2)
    const labels = [
      { text: 'to-b', end: 'to' as const },
      { text: 'to-a', end: 'from' as const },
    ]
    const measured = createRouting(DEFAULT_OPTIONS, undefined, TYPE)

    for (const [from, to] of [
      ['harbour', 'lighthouse'],
      ['lighthouse', 'harbour'],
    ] as const) {
      const [toB, toA] = routeEdge(
        { from, to, labels, arrows: ['to', 'from'] },
        byId,
        measured,
      )!.titles
      // The record pointing at `to` is near the `to` end and reads toward it, on the left of its reading.
      expect([toB!.heading, toB!.side, toB!.anchor]).toEqual(['along', -1, end])
      expect([toA!.heading, toA!.side, toA!.anchor]).toEqual(['against', 1, start])
    }
  })

  it('sets two labels by the notes that wrote them where one record has no head', () => {
    const [start, end] = getLabelPositions(2)
    const labels = [{ text: 'guides', end: 'to' as const }, { text: 'shelters' }]
    const [first, second] = routeEdge(
      { from: 'harbour', to: 'lighthouse', labels },
      byId,
      createRouting(DEFAULT_OPTIONS),
    )!.titles
    expect([first!.anchor, second!.anchor]).toEqual([start, end])
    expect([first!.heading, second!.heading]).toEqual(['along', 'against'])
  })

  it('reads the one label of a line along the way that is readable, centered and on the line', () => {
    const measured = createRouting(DEFAULT_OPTIONS, undefined, TYPE)
    const edge = { from: 'lighthouse', to: 'harbour', labels: [{ text: 'guides' }] }
    const [only] = routeEdge(edge, byId, measured)!.titles

    expect([only!.heading, only!.side, only!.shift]).toEqual(['against', 0, 0])
    expect(only!.rise).toBe(TYPE.xHeight / 2)
    expect(only!.anchor).toBe(MIDDLE)
  })

  it('draws no title for a label with no words', () => {
    expect(getPlaces({ from: 'harbour', to: 'lighthouse', labels: [{ text: '' }] })).toEqual([])
  })

  it('shares the room of the line between two labels', () => {
    const measure = (words: string) => 4 * [...words].length
    const long = 'the long way round the headland'
    const routed = routeEdge(
      {
        from: 'harbour',
        to: 'lighthouse',
        labels: [{ text: long }, { text: long }],
        arrows: ['to', 'from'],
      },
      byId,
      createRouting(DEFAULT_OPTIONS, measure),
    )!
    const room = (getLength(routed) - 2 * DEFAULT_OPTIONS.routing.arrowClearance) / 2

    expect(routed.titles).toHaveLength(2)
    for (const title of routed.titles) expect(measure(title.words)).toBeLessThanOrEqual(room)
  })
})

/**
 * Boxes too close for a straight line to carry an arrowhead and a word are joined
 * by a bow out of their sides, so a line between two notes next to each other is
 * drawn as fully as any other.
 */
describe('a line between two boxes with no room between them', () => {
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

  const measure = (words: string) => 6 * [...words].length
  const withWords = createRouting(DEFAULT_OPTIONS, measure)

  const route = (from: PlacedNode, to: PlacedNode, edge: Partial<PlexEdge> = {}) =>
    routeEdge(
      { from: from.id, to: to.id, ...edge },
      new Map([from, to].map((node) => [node.id, node])),
      withWords,
    )!

  it('bows out of the top of two boxes in a row below the focus, towards the focus', () => {
    const edge = route(box('a', 'child', 0, 200), box('b', 'child', 116, 200))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 180 })
    expect(edge.toPoint).toStrictEqual({ x: 116, y: 180 })
    expect(edge.control1.y).toBeLessThan(180)
    expect(edge.control2.y).toBeLessThan(180)
  })

  it('bows out of the bottom of two boxes in a row above the focus, towards the focus', () => {
    const edge = route(box('a', 'parent', 0, -200), box('b', 'parent', 116, -200))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: -180 })
    expect(edge.control1.y).toBeGreaterThan(-180)
  })

  it('bows out of the right of two boxes in a column right of the focus, away from it', () => {
    const edge = route(box('a', 'sibling', 300, 0), box('b', 'sibling', 300, 56))

    expect(edge.fromPoint).toStrictEqual({ x: 350, y: 0 })
    expect(edge.toPoint).toStrictEqual({ x: 350, y: 56 })
    expect(edge.control1.x).toBeGreaterThan(350)
  })

  it('bows out of the left of two boxes in a column left of the focus, away from it', () => {
    const edge = route(box('a', 'jump', -300, 0), box('b', 'jump', -300, 56))

    expect(edge.fromPoint).toStrictEqual({ x: -350, y: 0 })
    expect(edge.control1.x).toBeLessThan(-350)
  })

  it('keeps the ends of the edge the caller gave, whichever box is first', () => {
    const edge = route(box('b', 'child', 116, 200), box('a', 'child', 0, 200))

    expect(edge.fromPoint).toStrictEqual({ x: 116, y: 180 })
    expect(edge.toPoint).toStrictEqual({ x: 0, y: 180 })
  })

  it('has room for an arrowhead at each end and a title near each', () => {
    const edge = route(box('a', 'child', 0, 200), box('b', 'child', 116, 200), {
      labels: [{ text: 'onward' }, { text: 'back' }],
      arrows: ['to', 'from'],
    })

    expect(edge.arrowheads).toHaveLength(2)
    expect(edge.titles.map((title) => title.words)).toEqual(['onward', 'back'])
  })

  it('leaves a straight line where there is room for one', () => {
    const edge = route(box('a', 'child', 0, 200), box('b', 'child', 200, 200))

    expect(edge.fromPoint).toStrictEqual({ x: 50, y: 200 })
    expect(edge.toPoint).toStrictEqual({ x: 150, y: 200 })
  })

  it('leaves the edges at the focus the space the focus is given', () => {
    const edge = route(box('focus', 'focus', 0, 0), box('child', 'child', 0, 50))

    expect(edge.fromPoint).toStrictEqual({ x: 0, y: 20 })
    expect(edge.toPoint).toStrictEqual({ x: 0, y: 30 })
  })
})

describe('the curvature of a line', () => {
  const offset: PlexNeighborhood = {
    nodes: [
      { id: 'focus', title: 'Ports', relation: 'focus' },
      { id: 'child-0', title: 'Domain', relation: 'child' },
      { id: 'child-1', title: 'Adapter', relation: 'child' },
      { id: 'child-2', title: 'Use case', relation: 'child' },
    ],
    edges: [
      { from: 'focus', to: 'child-0' },
      { from: 'focus', to: 'child-1' },
      { from: 'focus', to: 'child-2' },
    ],
  }

  const getSidewaysEdge = (curvature: number) => {
    const frame = arrangePlex(offset, { options: { routing: { curvature } } })
    const child = getNodesByRelation(frame, 'child').reduce((a, b) =>
      Math.abs(a.x) > Math.abs(b.x) ? a : b,
    )
    expect(Math.abs(child.x)).toBeGreaterThan(0)
    return frame.edges.find((edge) => edge.to === child.id)!
  }

  /** Whether a point lies on the segment between two others. */
  const isOnSegment = (
    point: { x: number; y: number },
    start: { x: number; y: number },
    end: { x: number; y: number },
  ): boolean => {
    const cross = (end.x - start.x) * (point.y - start.y) - (end.y - start.y) * (point.x - start.x)
    const dot = (point.x - start.x) * (end.x - start.x) + (point.y - start.y) * (end.y - start.y)
    const length = (end.x - start.x) ** 2 + (end.y - start.y) ** 2
    return Math.abs(cross) < 1e-6 && dot >= -1e-6 && dot <= length + 1e-6
  }

  it('draws a straight line between two notes at curvature 0', () => {
    const edge = getSidewaysEdge(0)

    expect(isOnSegment(edge.control1, edge.fromPoint, edge.toPoint)).toBe(true)
    expect(isOnSegment(edge.control2, edge.fromPoint, edge.toPoint)).toBe(true)
  })

  it('bends the same line at the default curvature', () => {
    const edge = getSidewaysEdge(DEFAULT_OPTIONS.routing.curvature)

    expect(isOnSegment(edge.control1, edge.fromPoint, edge.toPoint)).toBe(false)
    expect(isOnSegment(edge.control2, edge.fromPoint, edge.toPoint)).toBe(false)
  })

  it('draws every line straight at curvature 0, whatever relation it runs along', () => {
    const frame = arrangePlex(neighborhoods.typical, { options: { routing: { curvature: 0 } } })
    const bent = frame.edges
      .filter(
        (edge) =>
          !isOnSegment(edge.control1, edge.fromPoint, edge.toPoint) ||
          !isOnSegment(edge.control2, edge.fromPoint, edge.toPoint),
      )
      .map((edge) => `${edge.from} ${edge.to}`)

    expect(bent).toStrictEqual([])
  })

  it('draws a line between two neighbors side by side straight at curvature 0', () => {
    const box = (id: string, x: number): PlacedNode => ({
      id,
      title: id,
      relation: 'child',
      x,
      y: 200,
      width: 100,
      height: 40,
      order: 0,
      opacity: 1,
    })
    const routing = createRouting(
      resolveOptions({ routing: { curvature: 0 } }),
      (words) => 6 * words.length,
    )
    const nodes = [box('a', 0), box('b', 116)]
    const edge = routeEdge(
      { from: 'a', to: 'b' },
      new Map(nodes.map((node) => [node.id, node])),
      routing,
    )!

    expect(isOnSegment(edge.control1, edge.fromPoint, edge.toPoint)).toBe(true)
    expect(isOnSegment(edge.control2, edge.fromPoint, edge.toPoint)).toBe(true)
  })

  it('draws the line between two linked children straight at curvature 0', () => {
    const linked: PlexNeighborhood = {
      nodes: [
        { id: 'focus', title: 'Ports', relation: 'focus' },
        { id: 'child-0', title: 'Domain', relation: 'child' },
        { id: 'child-1', title: 'Adapter', relation: 'child' },
      ],
      edges: [
        { from: 'focus', to: 'child-0' },
        { from: 'focus', to: 'child-1' },
        { from: 'child-0', to: 'child-1' },
      ],
    }
    const frame = arrangePlex(linked, {
      options: { routing: { curvature: 0 }, viewport: { width: 1200, height: 800 } },
      measure: (node) => 40 + node.title.length * 7,
    })
    const edge = frame.edges.find((one) => one.from === 'child-0' && one.to === 'child-1')!

    expect(edge).toBeDefined()
    expect(isOnSegment(edge.control1, edge.fromPoint, edge.toPoint)).toBe(true)
    expect(isOnSegment(edge.control2, edge.fromPoint, edge.toPoint)).toBe(true)
  })
})
