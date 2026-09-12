/** Where along its line a title comes to rest. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { MIDDLE, routeEdges, routingFor } from './routing'
import { settleTitles } from './titles'
import { build } from '../../fixtures/build'
import { neighbourhoods } from '../../fixtures/neighbourhoods'
import { headingOf, lengthOf, rulerOf, type PlacedEdge } from '../edge'
import type { PlexNeighbourhood } from '../neighbourhood'
import type { PlacedNode } from '../node'

/** A letter of a fixed width, so what a title needs is a matter of counting. */
const perLetter = (width: number) => (label: string) => width * [...label].length

const measureLabel = perLetter(6)

/** How deep a line of label type stands, as the window would have measured it. */
const labelDepth = 12

/** What every arrangement here measures its titles with. */
const measured = { measureLabel, labelDepth }

const VIEWPORT = { width: 1200, height: 800 }

/**
 * A window narrow enough that a row runs its titles short of room. What a
 * title does when the line it is set on cannot hold it is only visible where
 * some of them cannot.
 */
const CROWDED = { width: 900, height: 800 }

/**
 * How far along its line a title has moved when all it did was step clear of
 * the boxes its own line joins: a nudge, and not the slide of a title that had
 * to go looking for room.
 */
const A_NUDGE = 0.1

interface Box {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

const meets = (one: Box, other: Box): boolean =>
  one.minX < other.maxX &&
  other.minX < one.maxX &&
  one.minY < other.maxY &&
  other.minY < one.maxY

/**
 * Where a title's words lie in the picture: the run of them along the line,
 * with half a line of type standing either side of the line.
 */
function boxOf(edge: PlacedEdge, width: (label: string) => number): Box {
  const arc = lengthOf(edge)
  const along = rulerOf(edge)
  const at = edge.heading === 'against' ? 1 - edge.wordsAt : edge.wordsAt
  const half = width(edge.words!) / 2 / arc
  const deep = labelDepth / 2

  const run = [0, 0.25, 0.5, 0.75, 1].map((step) => along(at - half + 2 * half * step))
  const box = { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity }

  for (const [sample, point] of run.entries()) {
    const back = run[Math.max(sample - 1, 0)]!
    const on = run[Math.min(sample + 1, run.length - 1)]!
    const span = Math.hypot(on.x - back.x, on.y - back.y) || 1
    const acrossX = (Math.abs(on.y - back.y) / span) * deep
    const acrossY = (Math.abs(on.x - back.x) / span) * deep

    box.minX = Math.min(box.minX, point.x - acrossX)
    box.minY = Math.min(box.minY, point.y - acrossY)
    box.maxX = Math.max(box.maxX, point.x + acrossX)
    box.maxY = Math.max(box.maxY, point.y + acrossY)
  }
  return box
}

const boxAround = (node: PlacedNode): Box => ({
  minX: node.x - node.width / 2,
  minY: node.y - node.height / 2,
  maxX: node.x + node.width / 2,
  maxY: node.y + node.height / 2,
})

/** One line, with room along it for whatever words it is given. */
const alone = (label: string): PlexNeighbourhood => ({
  nodes: [
    { id: 'focus', title: 'Here', seat: 'focus' },
    { id: 'aside', title: 'Aside', seat: 'jump' },
  ],
  edges: [{ from: 'aside', to: 'focus', label }],
})

describe('a title finds room on its line', () => {
  it('keeps the middle of a line with nothing near it', () => {
    const frame = arrangePlex(alone('see'), measured)

    expect(frame.edges[0]!.wordsAt).toBe(0.5)
  })

  it('gives up the middle to stay clear of the boxes its own line joins', () => {
    // Words nearly as long as the line they are set on: the middle would put
    // them against a box at either end.
    const frame = arrangePlex(alone('see also'), measured)

    expect(frame.edges[0]!.words).toBe('see also')
    expect(frame.edges[0]!.wordsAt).not.toBe(MIDDLE)
    expect(Math.abs(frame.edges[0]!.wordsAt - MIDDLE)).toBeLessThan(A_NUDGE)
  })

  it('stands every title of a fan clear of the others', () => {
    const fan = build('A node', { child: 5 })
    const frame = arrangePlex(fan, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.edges.map((edge) => boxOf(edge, measureLabel))

    expect(boxes.length).toBe(5)

    const piled: string[] = []
    for (let one = 0; one < boxes.length; one += 1) {
      for (let other = one + 1; other < boxes.length; other += 1) {
        if (meets(boxes[one]!, boxes[other]!)) piled.push(`${one} x ${other}`)
      }
    }
    expect(piled).toStrictEqual([])
  })

  it('slides the titles that are in the way, and leaves the rest at the middle', () => {
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const moved = frame.edges.filter((edge) => edge.wordsAt !== 0.5)

    expect(moved.length).toBeGreaterThan(0)
    expect(moved.length).toBeLessThan(frame.edges.length)
  })

  it('writes nothing at all on a line with nowhere clear along it', () => {
    const pair = alone('see also')
    const placed = arrangePlex(pair, measured)
    const byId = new Map(placed.nodes.map((node) => [node.id, node]))
    const routing = routingFor(DEFAULT_OPTIONS, measureLabel, labelDepth)
    const routed = routeEdges(pair.edges, byId, routing)

    // A box over the whole picture, so no place on the line is clear of it.
    const blanket: PlacedNode = {
      id: 'blanket',
      title: 'Blanket',
      seat: 'sibling',
      x: 0,
      y: 0,
      width: 2000,
      height: 2000,
      order: 0,
      opacity: 1,
    }
    const settled = settleTitles(routed, [...placed.nodes, blanket], routing)

    expect(settled[0]!.words).toBeUndefined()
  })

  it('gives the same neighbourhood the same offsets twice over', () => {
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const once = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const again = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    expect(again.edges.map((edge) => edge.wordsAt)).toStrictEqual(
      once.edges.map((edge) => edge.wordsAt),
    )
  })

  it('sets a title where the arrangement says, and not where the curve runs', () => {
    // A title read against its curve is set from the far end, so the fraction
    // it carries is a fraction of the line the words are read along.
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    for (const edge of frame.edges) {
      expect(edge.wordsAt, `${edge.from}->${edge.to}`).toBeGreaterThan(0)
      expect(edge.wordsAt, `${edge.from}->${edge.to}`).toBeLessThan(1)
    }
  })

  it('keeps a title clear of the boxes wherever the line lets it', () => {
    const crowd = build('A node', { parent: 2, jump: 3 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.nodes.map(boxAround)

    const over = frame.edges.filter((edge) =>
      boxes.some((box) => meets(box, boxOf(edge, measureLabel))),
    )
    expect(over).toStrictEqual([])
  })

  it('reads the words the way the line runs where they end up', () => {
    // A long line taken right to left, with a box over the middle of it. The
    // words are read from the far end, so the fraction the title carries is a
    // fraction of the line the words are read along and not of the curve.
    const far: PlacedNode = {
      id: 'far',
      title: 'Far',
      seat: 'jump',
      x: -300,
      y: 0,
      width: 100,
      height: 36,
      order: 0,
      opacity: 1,
    }
    const here: PlacedNode = { ...far, id: 'here', title: 'Here', seat: 'focus', x: 300 }
    const covering: PlacedNode = {
      ...far,
      id: 'covering',
      title: 'Covering',
      seat: 'sibling',
      x: 0,
      y: 40,
      width: 100,
      height: 100,
    }

    const byId = new Map([far, here].map((node) => [node.id, node]))
    const routing = routingFor(DEFAULT_OPTIONS, measureLabel, labelDepth)
    const routed = routeEdges([{ from: 'here', to: 'far', label: 'see also' }], byId, routing)
    expect(headingOf(routed[0]!, MIDDLE)).toBe('against')

    const settled = settleTitles(routed, [far, here, covering], routing)[0]!
    expect(settled.words).toBe('see also')
    expect(settled.heading).toBe('against')
    expect(settled.wordsAt).not.toBe(MIDDLE)

    // The place on the curve is the fraction taken the other way round.
    expect(headingOf(settled, 1 - settled.wordsAt)).toBe('against')
  })

  it('keeps a title off the boxes its line crosses', () => {
    const frame = arrangePlex(neighbourhoods.labelledRows, {
      options: { viewport: VIEWPORT },
      ...measured,
    })
    const boxes = frame.nodes.map(boxAround)

    const over = frame.edges
      .filter((edge) => edge.words)
      .filter((edge) => boxes.some((box) => meets(box, boxOf(edge, measureLabel))))
      .map((edge) => edge.words)

    expect(over).toStrictEqual([])
  })

  it('cuts a title to the room its line has left, or writes none of it', () => {
    const frame = arrangePlex(neighbourhoods.labelledRows, {
      options: { viewport: CROWDED },
      ...measured,
    })
    const named = neighbourhoods.labelledRows.edges.filter((edge) => edge.label)
    const written = frame.edges.filter((edge) => edge.words)

    // Some of them are there whole, and every one that is there is either the
    // whole label or a cut of its start.
    expect(written.length).toBeGreaterThan(0)
    expect(written.length).toBeLessThan(named.length)
    for (const edge of written) {
      const label = named.find((one) => one.from === edge.from && one.to === edge.to)!
      expect(label.label!.startsWith(edge.words!.replace(/…$/, ''))).toBe(true)
    }
  })

  it('gives the line with the least room to spare its place first', () => {
    // Two lines crossing one band: the tight one takes the middle it needs,
    // and the roomy one goes round it.
    const crossing = arrangePlex(neighbourhoods.labelledRows, {
      options: { viewport: CROWDED },
      ...measured,
    })
    // Its line is barely longer than its words, and it keeps the middle of it.
    const tightest = crossing.edges.find((edge) => edge.to === 'child-1')!
    expect(tightest.words!.startsWith('the minutes of the me')).toBe(true)
    expect(Math.abs(tightest.wordsAt - MIDDLE)).toBeLessThan(A_NUDGE)

    // The line to the row beyond has room to spare, and gives way.
    const roomier = crossing.edges.find((edge) => edge.to === 'child-4')!
    expect(roomier.words === undefined || roomier.wordsAt !== MIDDLE).toBe(true)
  })

  it('leaves every title at the middle where nothing measured the words', () => {
    const crowd = build('A node', { child: 5 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT } })

    expect(frame.edges.map((edge) => edge.wordsAt)).toStrictEqual(
      frame.edges.map(() => 0.5),
    )
  })
})
