/** Where along its line a title evaluates to rest. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { DEFAULT_OPTIONS } from './options'
import { MIDDLE, routeEdges, createRouting } from './routing'
import { separateTitles } from './titles'
import { build } from '../../fixtures/build'
import { neighbourhoods } from '../../fixtures/neighbourhoods'
import { getHeading, getLength, createPointOnCurve, type PlacedEdge } from '../edge'
import type { PlexNeighbourhood } from '../neighbourhood'
import type { PlacedNode, Position } from '../node'

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

const isOverlapping = (one: Box, other: Box): boolean =>
  one.minX < other.maxX && other.minX < one.maxX && one.minY < other.maxY && other.minY < one.maxY

/**
 * Where a title's words lie in the picture: the run of them along the line,
 * with half a line of type standing either side of the line.
 */
function getBox(edge: PlacedEdge, width: (label: string) => number): Box {
  const arc = getLength(edge)
  const along = createPointOnCurve(edge)
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
const createOneLine = (label: string): PlexNeighbourhood => ({
  nodes: [
    { id: 'focus', title: 'Here', relation: 'focus' },
    { id: 'aside', title: 'Aside', relation: 'jump' },
  ],
  edges: [{ from: 'aside', to: 'focus', label }],
})

describe('a title finds room on its line', () => {
  it('keeps the middle of a line with nothing near it', () => {
    const frame = arrangePlex(createOneLine('see'), measured)

    expect(frame.edges[0]!.wordsAt).toBe(0.5)
  })

  it('gives up the middle to stay clear of the boxes its own line joins', () => {
    // Words nearly as long as the line they are set on: the middle would put
    // them against a box at either end.
    const frame = arrangePlex(createOneLine('see also'), measured)

    expect(frame.edges[0]!.words).toBe('see also')
    expect(frame.edges[0]!.wordsAt).not.toBe(MIDDLE)
    expect(Math.abs(frame.edges[0]!.wordsAt - MIDDLE)).toBeLessThan(A_NUDGE)
  })

  it('remains every title of a fan clear of the others', () => {
    const fan = build('A node', { child: 5 })
    const frame = arrangePlex(fan, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.edges.map((edge) => getBox(edge, measureLabel))

    expect(boxes.length).toBe(5)

    const piled: string[] = []
    for (let one = 0; one < boxes.length; one += 1) {
      for (let other = one + 1; other < boxes.length; other += 1) {
        if (isOverlapping(boxes[one]!, boxes[other]!)) piled.push(`${one} x ${other}`)
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
    const pair = createOneLine('see also')
    const placed = arrangePlex(pair, measured)
    const byId = new Map(placed.nodes.map((node) => [node.id, node]))
    const routing = createRouting(DEFAULT_OPTIONS, measureLabel, labelDepth)
    const routed = routeEdges(pair.edges, byId, routing)

    // A box over the whole picture, so no place on the line is clear of it.
    const blanket: PlacedNode = {
      id: 'blanket',
      title: 'Blanket',
      relation: 'sibling',
      x: 0,
      y: 0,
      width: 2000,
      height: 2000,
      order: 0,
      opacity: 1,
    }
    const settled = separateTitles(routed, [...placed.nodes, blanket], routing)

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
      boxes.some((box) => isOverlapping(box, getBox(edge, measureLabel))),
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
      relation: 'jump',
      x: -300,
      y: 0,
      width: 100,
      height: 36,
      order: 0,
      opacity: 1,
    }
    const here: PlacedNode = { ...far, id: 'here', title: 'Here', relation: 'focus', x: 300 }
    const covering: PlacedNode = {
      ...far,
      id: 'covering',
      title: 'Covering',
      relation: 'sibling',
      x: 0,
      y: 40,
      width: 100,
      height: 100,
    }

    const byId = new Map([far, here].map((node) => [node.id, node]))
    const routing = createRouting(DEFAULT_OPTIONS, measureLabel, labelDepth)
    const routed = routeEdges([{ from: 'here', to: 'far', label: 'see also' }], byId, routing)
    expect(getHeading(routed[0]!, MIDDLE)).toBe('against')

    const settled = separateTitles(routed, [far, here, covering], routing)[0]!
    expect(settled.words).toBe('see also')
    expect(settled.heading).toBe('against')
    expect(settled.wordsAt).not.toBe(MIDDLE)

    // The place on the curve is the fraction taken the other way round.
    expect(getHeading(settled, 1 - settled.wordsAt)).toBe('against')
  })

  it('keeps a title off the boxes its line crosses', () => {
    const frame = arrangePlex(neighbourhoods.labelledRows, {
      options: { viewport: VIEWPORT },
      ...measured,
    })
    const boxes = frame.nodes.map(boxAround)

    const over = frame.edges
      .filter((edge) => edge.words)
      .filter((edge) => boxes.some((box) => isOverlapping(box, getBox(edge, measureLabel))))
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

    expect(frame.edges.map((edge) => edge.wordsAt)).toStrictEqual(frame.edges.map(() => 0.5))
  })
})

/**
 * Lines drawn straight across a plain page, so where a title stands is a
 * matter of counting along them. A letter is ten wide, a line of type twelve
 * deep, and a title keeps a quarter of that clear of whatever it stands near.
 */
describe('a title set on a straight line', () => {
  const TEN = perLetter(10)
  const routing = createRouting(DEFAULT_OPTIONS, TEN, labelDepth)

  /** One step of the scan a line is looked at by, along the line. */
  const A_STEP = 3

  const createLine = (
    from: Position,
    to: Position,
    words: string,
    extra: Partial<PlacedEdge> = {},
  ): PlacedEdge => {
    const at = (share: number): Position => ({
      x: from.x + (to.x - from.x) * share,
      y: from.y + (to.y - from.y) * share,
    })
    return {
      from: 'from',
      to: 'to',
      label: words,
      opacity: 1,
      heading: to.x < from.x ? 'against' : 'along',
      words,
      wordsAt: MIDDLE,
      fromPoint: from,
      control1: at(1 / 3),
      control2: at(2 / 3),
      toPoint: to,
      ...extra,
    } as PlacedEdge
  }

  /** A box over the page, as far as a title is concerned. */
  const createBlocker = (id: string, left: number, right: number, y = 0): PlacedNode => ({
    id,
    title: id,
    relation: 'sibling',
    x: (left + right) / 2,
    y,
    width: right - left - 2 * 3,
    height: 20,
    order: 0,
    opacity: 1,
  })

  /** Where the middle of a title is, in the picture. */
  const getCentre = (edge: PlacedEdge): Position =>
    createPointOnCurve(edge)(edge.heading === 'against' ? 1 - edge.wordsAt : edge.wordsAt)

  const ACROSS = { from: { x: -200, y: 0 }, to: { x: 200, y: 0 } }
  const across = (words: string) => createLine(ACROSS.from, ACROSS.to, words)

  it('keeps clear of a box that remains over only the depth of the type', () => {
    // The box starts 6 below the line. The type reaches 9 either side of it, and
    // 3 of that is the room a title keeps from whatever it stands near.
    const lower = createBlocker('lower', -32.5, 13.5, 19)
    const [settled] = separateTitles([across('abcde')], [lower], routing)

    expect(getCentre(settled!).x - 25).toBeGreaterThanOrEqual(13.5)
    expect(getCentre(settled!).x - 25).toBeLessThan(13.5 + A_STEP + 0.01)
  })

  it('remains at the clear place nearest the middle, to the right of a box', () => {
    const [settled] = separateTitles(
      [across('abcde')],
      [createBlocker('box', -32.5, 13.5)],
      routing,
    )

    expect(settled!.words).toBe('abcde')
    expect(getCentre(settled!).x - 25).toBeGreaterThanOrEqual(13.5)
    expect(getCentre(settled!).x - 25).toBeLessThan(13.5 + A_STEP + 0.01)
    expect(settled!.wordsAt).toBeCloseTo(0.6025, 3)
  })

  it('remains at the clear place nearest the middle, to the left of a box', () => {
    const [settled] = separateTitles(
      [across('abcde')],
      [createBlocker('box', -13.5, 32.5)],
      routing,
    )

    expect(settled!.words).toBe('abcde')
    expect(getCentre(settled!).x + 25).toBeLessThanOrEqual(-13.5)
    expect(getCentre(settled!).x + 25).toBeGreaterThan(-13.5 - A_STEP - 0.01)
    expect(settled!.wordsAt).toBeCloseTo(0.4025, 3)
  })

  it('counts the fraction a title is read from along the line it is read on', () => {
    // The same line taken right to left, so the title is read from its far end.
    const backwards = createLine(ACROSS.to, ACROSS.from, 'abcde')
    const [settled] = separateTitles([backwards], [createBlocker('box', -32.5, 13.5)], routing)

    expect(settled!.heading).toBe('against')
    expect(settled!.wordsAt).toBeCloseTo(0.5975, 3)
    expect(getCentre(settled!).x - 25).toBeGreaterThanOrEqual(13.5)
  })

  it('starts a title at the near end of the only clear place that is to the right of the middle', () => {
    // A box over the middle and the whole left side of the line.
    const [settled] = separateTitles([across('abcde')], [createBlocker('box', -204, 53)], routing)

    expect(settled!.words).toBe('abcde')
    expect(getCentre(settled!).x - 25).toBeGreaterThanOrEqual(53)
    expect(getCentre(settled!).x - 25).toBeLessThan(53 + A_STEP + 0.01)
  })

  it('clears the titles already set by what they take up, and no more', () => {
    // Two lines a line of type apart: the first keeps the middle, and the
    // second goes round it by the nearer side.
    const upper = across('abcde')
    const lower = createLine({ x: -192, y: 10 }, { x: 208, y: 10 }, 'abcde')
    const [first, second] = separateTitles([upper, lower], [], routing)

    expect(first!.wordsAt).toBe(MIDDLE)
    expect(getCentre(second!).x - 25).toBeGreaterThanOrEqual(25)
    expect(getCentre(second!).x - 25).toBeLessThan(25 + A_STEP + 0.01)
  })

  it('sets the line with the least room first, whatever order they come in', () => {
    // The roomy line comes first, and its title would take the middle that the
    // tight line, barely longer than its words, has no other place for.
    const roomy = createLine({ x: -45, y: 0 }, { x: 45, y: 0 }, 'abcde')
    const tight = createLine({ x: -40, y: 10 }, { x: 40, y: 10 }, 'abcdefg')
    const [settledRoomy, settledTight] = separateTitles([roomy, tight], [], routing)

    expect(settledTight!.words).toBe('abcdefg')
    expect(settledTight!.wordsAt).toBeCloseTo(MIDDLE, 5)
    expect(settledRoomy!.words).toBeUndefined()
  })

  it('leaves a title where it was put when its words measure nothing', () => {
    const placed = createLine(ACROSS.from, ACROSS.to, 'abcde', { wordsAt: 0.3 })
    const [settled] = separateTitles(
      [placed],
      [],
      createRouting(DEFAULT_OPTIONS, () => 0, labelDepth),
    )

    expect(settled!.words).toBe('abcde')
    expect(settled!.wordsAt).toBe(0.3)
  })

  it('leaves a title where it was put when its line has no length', () => {
    const point = { x: 0, y: 0 }
    const placed = createLine(point, point, 'abcde', { wordsAt: 0.3 })
    const [settled] = separateTitles([placed], [], routing)

    expect(settled!.words).toBe('abcde')
    expect(settled!.wordsAt).toBe(0.3)
  })

  it('cuts a title to the longest clear place when no place holds the whole of it', () => {
    // Three places of about 60, 105 and 80 between boxes, for words 120 long.
    const boxes = [
      createBlocker('one', -140, -122),
      createBlocker('two', -17, 0),
      createBlocker('three', 80, 200),
    ]
    const [settled] = separateTitles([across('abcdefghijkl')], boxes, routing)

    expect(settled!.words).toBe('abcdefghi…')
    expect(getCentre(settled!).x).toBeGreaterThan(-122)
    expect(getCentre(settled!).x).toBeLessThan(-17)
  })

  it('keeps the room an arrowhead takes at the end of the line', () => {
    // Clear only from 136 to the end, which an arrowhead takes 20 of.
    const arrow = createLine(ACROSS.from, ACROSS.to, 'abcdefgh', { arrow: 'to' })
    const [settled] = separateTitles([arrow], [createBlocker('box', -204, 134)], routing)

    expect(settled!.words).toBe('abc…')
    expect(getCentre(settled!).x + TEN(settled!.words!) / 2).toBeLessThanOrEqual(180)
  })

  it('writes no title on a line that an arrowhead takes the whole of', () => {
    const short = createLine({ x: 0, y: 0 }, { x: 30, y: 0 }, 'a', { arrow: 'to' })
    const [settled] = separateTitles([short], [], routing)

    expect(settled!.words).toBeUndefined()
  })

  it('keeps a title clear of the corner of a slanting title that a box around it would take', () => {
    // The first title slants across the page; the second line runs below it,
    // under the empty corner of the box around the first.
    const slanting = createLine({ x: -100, y: -100 }, { x: 100, y: 100 }, 'abcde')
    const below = createLine({ x: -200, y: 28 }, { x: 200, y: 28 }, 'abcde')
    const [, second] = separateTitles([slanting, below], [], routing)

    expect(getCentre(second!).x).toBeGreaterThan(-30)
    expect(getCentre(second!).x).toBeLessThan(-10)
  })
})
