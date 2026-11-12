/** Where along its line a title evaluates to rest. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from '../arrange'
import { DEFAULT_OPTIONS } from '../options'
import {
  MIDDLE,
  createTitle,
  getLabelPlaces,
  type LabelPlace,
  getDepth,
  getRise,
  getShift,
  routeEdges,
  createRouting,
} from './routing'
import { separateTitles } from './titles'
import { runBoxes } from '../layout/boxes'
import { build } from '../../../fixtures/build'
import { graphs } from '../../../fixtures/graphs'
import { getHeading, getLength, createPointOnCurve, type PlacedEdge } from '../../edge'
import type { PlacedNode, Position } from '../../node'
import type { EdgeArrow, EdgeLabel, PlexGraph } from '../../../../types'

const perLetter = (width: number) => (label: string) => width * [...label].length

const measureLabel = perLetter(6)

/** The ink of the label font, as measured in the window. */
const labelType = { ascent: 9, descent: 3, xHeight: 6 }

/** The height of a line of label text. */
const labelDepth = getDepth(labelType)

/** What every arrangement here measures its titles with. */
const measured = { measureLabel, labelType }

const setting = ({ side, heading }: LabelPlace) => ({
  side,
  shift: getShift(side, labelType),
  rise: getRise(side, labelType),
  ...(heading ? { heading } : {}),
})

const VIEWPORT = { width: 1200, height: 800 }

/**
 * A narrow layout in which a row of titles has too little space. Title
 * behavior on a line too short to hold it is only observable where some titles
 * do not fit.
 */
const CROWDED = { width: 900, height: 800 }

/**
 * The offset along its line of a title that only moved clear of the boxes its
 * own line joins: a nudge, not the slide of a title that had to find space.
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

function getBox(edge: PlacedEdge, width: (label: string) => number): Box {
  const arc = getLength(edge)
  const axisEdge = createPointOnCurve(edge)
  const offset =
    edge.titles[0]!.heading === 'against' ? 1 - edge.titles[0]!.wordsAt : edge.titles[0]!.wordsAt
  const half = width(edge.titles[0]!.words) / 2 / arc
  const deep = labelDepth / 2

  const run = [0, 0.25, 0.5, 0.75, 1].map((step) => axisEdge(offset - half + 2 * half * step))
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

const createOneLine = (label: string): PlexGraph => ({
  nodes: [
    { id: 'focus', title: 'Here', relation: 'focus' },
    { id: 'aside', title: 'Aside', relation: 'jump' },
  ],
  edges: [{ from: 'aside', to: 'focus', labels: [{ text: label }] }],
})

describe('a title finds room on its line', () => {
  it('keeps the middle of a line with nothing near it', () => {
    const frame = arrangePlex(createOneLine('see'), measured)

    expect(frame.edges[0]!.titles[0]!.wordsAt).toBe(0.5)
  })

  it('gives up the middle to stay clear of the boxes its own line joins', () => {
    // Words nearly as long as the line they are set on: the middle would put
    // them against a box at either end.
    const frame = arrangePlex(createOneLine('see also'), measured)

    expect(frame.edges[0]!.titles[0]?.words).toBe('see also')
    expect(frame.edges[0]!.titles[0]!.wordsAt).not.toBe(MIDDLE)
    expect(Math.abs(frame.edges[0]!.titles[0]!.wordsAt - MIDDLE)).toBeLessThan(A_NUDGE)
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
    const moved = frame.edges.filter((edge) => edge.titles[0]!.wordsAt !== 0.5)

    expect(moved.length).toBeGreaterThan(0)
    expect(moved.length).toBeLessThan(frame.edges.length)
  })

  it('writes nothing at all on a line with nowhere clear along it', () => {
    const pair = createOneLine('see also')
    const placedLine = arrangePlex(pair, measured)
    const byId = new Map(placedLine.nodes.map((node) => [node.id, node]))
    const edgeRouting = createRouting(DEFAULT_OPTIONS, measureLabel, labelType)
    const routed = routeEdges(pair.edges, byId, edgeRouting)

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
    const resolved = separateTitles(routed, [...placedLine.nodes, blanket], edgeRouting)

    expect(resolved[0]!.titles[0]?.words).toBeUndefined()
  })

  it('gives the same graph the same offsets twice over', () => {
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const once = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const again = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    expect(again.edges.map((edge) => edge.titles[0]!.wordsAt)).toStrictEqual(
      once.edges.map((edge) => edge.titles[0]!.wordsAt),
    )
  })

  it('sets a title where the arrangement says, and not where the curve runs', () => {
    // A title read against its curve is set from the far end, so the fraction
    // it carries is a fraction of the line the words are read along.
    const crowd = build('A node', { parent: 2, child: 6, jump: 3, sibling: 2 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })

    for (const edge of frame.edges) {
      expect(edge.titles[0]!.wordsAt, `${edge.from}->${edge.to}`).toBeGreaterThan(0)
      expect(edge.titles[0]!.wordsAt, `${edge.from}->${edge.to}`).toBeLessThan(1)
    }
  })

  it('keeps a title clear of the boxes wherever the line lets it', () => {
    const crowd = build('A node', { parent: 2, jump: 3 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT }, ...measured })
    const boxes = frame.nodes.map(boxAround)

    const overEdges = frame.edges.filter((edge) =>
      boxes.some((box) => isOverlapping(box, getBox(edge, measureLabel))),
    )
    expect(overEdges).toStrictEqual([])
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
    const edgeRouting = createRouting(DEFAULT_OPTIONS, measureLabel, labelType)
    const routed = routeEdges(
      [{ from: 'here', to: 'far', labels: [{ text: 'see also' }] }],
      byId,
      edgeRouting,
    )
    expect(getHeading(routed[0]!, MIDDLE)).toBe('against')

    const resolved = separateTitles(routed, [far, here, covering], edgeRouting)[0]!
    expect(resolved.titles[0]?.words).toBe('see also')
    expect(resolved.titles[0]!.heading).toBe('against')
    expect(resolved.titles[0]!.wordsAt).not.toBe(MIDDLE)

    // The place on the curve is the fraction taken the other way round.
    expect(getHeading(resolved, 1 - resolved.titles[0]!.wordsAt)).toBe('against')
  })

  it('keeps a title off the boxes its line crosses', () => {
    const frame = arrangePlex(graphs.labelledRows, {
      options: { viewport: VIEWPORT },
      ...measured,
    })
    const boxes = frame.nodes.map(boxAround)

    const overEdges = frame.edges
      .filter((edge) => edge.titles[0]?.words)
      .filter((edge) => boxes.some((box) => isOverlapping(box, getBox(edge, measureLabel))))
      .map((edge) => edge.titles[0]?.words)

    expect(overEdges).toStrictEqual([])
  })

  it('cuts a title to the room its line has left, or writes none of it', () => {
    const frame = arrangePlex(graphs.labelledRows, {
      options: { viewport: CROWDED },
      ...measured,
    })
    const named = graphs.labelledRows.edges.filter((edge) => edge.labels?.[0]?.text)
    const content = frame.edges.filter((edge) => edge.titles[0]?.words)

    // Some of them are there whole, and every one that is there is either the
    // whole label or a cut of its start.
    expect(content.length).toBeGreaterThan(0)
    expect(content.length).toBeLessThan(named.length)
    for (const edge of content) {
      const label = named.find((one) => one.from === edge.from && one.to === edge.to)!
      expect(label.labels![0]!.text.startsWith(edge.titles[0]!.words.replace(/…$/, ''))).toBe(true)
    }
  })

  it('gives the line with the least room to spare its place first', () => {
    // Two lines crossing one band: the tight one takes the middle it needs,
    // and the roomy one goes round it.
    const crossing = arrangePlex(graphs.labelledRows, {
      options: { viewport: CROWDED },
      ...measured,
    })
    // Its line is barely longer than its words, and it keeps the middle of it.
    const tightest = crossing.edges.find((edge) => edge.to === 'child-1')!
    expect(tightest.titles[0]!.words.startsWith('the minutes of the me')).toBe(true)
    expect(Math.abs(tightest.titles[0]!.wordsAt - MIDDLE)).toBeLessThan(A_NUDGE)

    // The line to the farther row has spare length and yields.
    const roomier = crossing.edges.find((edge) => edge.to === 'child-4')!
    expect(roomier.titles[0]?.words === undefined || roomier.titles[0]!.wordsAt !== MIDDLE).toBe(
      true,
    )
  })

  it('leaves every title at the middle where nothing measured the words', () => {
    const crowd = build('A node', { child: 5 })
    const frame = arrangePlex(crowd, { options: { viewport: VIEWPORT } })

    expect(frame.edges.map((edge) => edge.titles[0]!.wordsAt)).toStrictEqual(
      frame.edges.map(() => 0.5),
    )
  })
})

/**
 * The two labels of one line: one on each side of it, each read in the
 * direction of its record, kept in its own half and clear of the other.
 */
describe('two titles set on opposite sides of a line', () => {
  const TEN = perLetter(10)
  const edgeRouting = createRouting(DEFAULT_OPTIONS, TEN, labelType)
  const ANGLES = [0, 30, 90, 135, 180, 270]

  const createPair = (angle: number): PlacedEdge => {
    const radians = (angle * Math.PI) / 180
    const from = { x: 0, y: 0 }
    const to = { x: 400 * Math.cos(radians), y: 400 * Math.sin(radians) }
    const pointAt = (share: number): Position => ({ x: to.x * share, y: to.y * share })
    const curve = {
      fromPoint: from,
      control1: pointAt(1 / 3),
      control2: pointAt(2 / 3),
      toPoint: to,
    }
    const labels = [{ text: 'guides' }, { text: 'shelters' }]
    const [first, second] = getLabelPlaces(labels)
    return {
      from: 'from',
      to: 'to',
      labels: [{ text: 'guides' }, { text: 'shelters' }],
      opacity: 1,
      ...curve,
      titles: [
        createTitle(curve, first!.position, 'guides', setting(first!)),
        createTitle(curve, second!.position, 'shelters', setting(second!)),
      ],
    }
  }

  /**
   * Where a title is drawn on a line that starts at the origin: the fraction of the line it is
   * at, the box it fills, and the distance of the center of that box from the line, positive on
   * the right looking from the start of the line.
   */
  const getPlace = (edge: PlacedEdge, index: number) => {
    const title = edge.titles[index]!
    const fraction = title.heading === 'against' ? 1 - title.wordsAt : title.wordsAt
    const length = Math.hypot(edge.toPoint.x, edge.toPoint.y)
    const half = (title.words.length * 10) / 2 / length
    const box = runBoxes(edge, { depth: labelDepth, shift: title.shift })(
      fraction - half,
      fraction + half,
    )
    const center = { x: (box.minX + box.maxX) / 2, y: (box.minY + box.maxY) / 2 }
    const distance = (center.y * edge.toPoint.x - center.x * edge.toPoint.y) / length
    return { fraction, box, distance }
  }

  it.each(ANGLES)('puts them on opposite sides with the line between, at %i degrees', (angle) => {
    const [guides, shelters] = separateTitles([createPair(angle)], [], edgeRouting)[0]!.titles

    expect(guides!.side).toBe(-1)
    expect(shelters!.side).toBe(1)
    expect(guides!.shift).toBeLessThan(-labelDepth / 2)
    expect(shelters!.shift).toBeGreaterThan(labelDepth / 2)

    // Measured on the page: the boxes are on either side of the line, clear of it.
    const edge = separateTitles([createPair(angle)], [], edgeRouting)[0]!
    expect(getPlace(edge, 0).distance).toBeLessThan(-labelDepth / 2)
    expect(getPlace(edge, 1).distance).toBeGreaterThan(labelDepth / 2)
  })

  it.each(ANGLES)('reads each in the direction of its record at %i degrees', (angle) => {
    const [guides, shelters] = separateTitles([createPair(angle)], [], edgeRouting)[0]!.titles

    expect(guides!.heading).toBe('along')
    expect(shelters!.heading).toBe('against')
  })

  it.each(ANGLES)(
    'keeps them in their own halves and clear of each other at %i degrees',
    (angle) => {
      const edge = separateTitles([createPair(angle)], [], edgeRouting)[0]!
      const one = getPlace(edge, 0)
      const other = getPlace(edge, 1)

      expect(edge.titles).toHaveLength(2)
      expect(one.fraction).toBeLessThan(MIDDLE)
      expect(other.fraction).toBeGreaterThan(MIDDLE)
      expect(isOverlapping(one.box, other.box)).toBe(false)
    },
  )

  it('leaves the title of a line with one label on the line', () => {
    const edge = createPair(0)
    const centered = {
      ...edge,
      titles: [createTitle(edge, MIDDLE, 'guides', { side: 0, shift: 0, rise: 0 })],
    }
    const [title] = separateTitles([centered], [], edgeRouting)[0]!.titles

    expect([title!.side, title!.shift, title!.wordsAt]).toEqual([0, 0, MIDDLE])
  })

  it('keeps the side and the reading of a title that was moved along its line', () => {
    // A box over the start of the line pushes the first title on.
    const blocker: PlacedNode = {
      id: 'blocker',
      title: 'Blocker',
      relation: 'sibling',
      x: 80,
      y: 0,
      width: 100,
      height: 60,
      order: 0,
      opacity: 1,
    }
    const [guides] = separateTitles([createPair(0)], [blocker], edgeRouting)[0]!.titles

    expect(guides!.wordsAt).not.toBeCloseTo(0.2)
    expect([guides!.side, guides!.heading]).toEqual([-1, 'along'])
  })
})

/**
 * Straight lines across a plain canvas, so title positions follow from
 * counting along them. A letter is 10 wide, a line of text 12 high, and a title
 * keeps a quarter of that clear of nearby boxes.
 */
describe('a title set on a straight line', () => {
  const TEN = perLetter(10)
  const edgeRouting = createRouting(DEFAULT_OPTIONS, TEN, labelType)

  /** One step of the scan a line is looked at by, along the line. */
  const A_STEP = 3

  const createLine = (
    from: Position,
    to: Position,
    words: string,
    extra: Partial<PlacedEdge> & { readonly wordsAt?: number; readonly anchor?: number } = {},
  ): PlacedEdge => {
    const { wordsAt = MIDDLE, anchor = wordsAt, ...rest } = extra
    const pointAt = (share: number): Position => ({
      x: from.x + (to.x - from.x) * share,
      y: from.y + (to.y - from.y) * share,
    })
    return {
      from: 'from',
      to: 'to',
      labels: [{ text: words }],
      opacity: 1,
      titles: [
        {
          anchor,
          heading: to.x < from.x ? 'against' : 'along',
          side: 0,
          shift: 0,
          rise: 0,
          words,
          wordsAt,
        },
      ],
      fromPoint: from,
      control1: pointAt(1 / 3),
      control2: pointAt(2 / 3),
      toPoint: to,
      ...rest,
    }
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

  /** The center of a title. */
  const getCenter = (edge: PlacedEdge): Position =>
    createPointOnCurve(edge)(
      edge.titles[0]!.heading === 'against' ? 1 - edge.titles[0]!.wordsAt : edge.titles[0]!.wordsAt,
    )

  const ACROSS = { from: { x: -200, y: 0 }, to: { x: 200, y: 0 } }
  const acrossLine = (words: string) => createLine(ACROSS.from, ACROSS.to, words)

  it('keeps clear of a box that remains over only the depth of the type', () => {
    // The box starts 6 below the line. The text extends 9 on either side of it,
    // and 3 of that is the clearance a title keeps from nearby boxes.
    const lower = createBlocker('lower', -32.5, 13.5, 19)
    const [resolved] = separateTitles([acrossLine('abcde')], [lower], edgeRouting)

    expect(getCenter(resolved!).x - 25).toBeGreaterThanOrEqual(13.5)
    expect(getCenter(resolved!).x - 25).toBeLessThan(13.5 + A_STEP + 0.01)
  })

  it('remains at the clear place nearest the middle, to the right of a box', () => {
    const [resolved] = separateTitles(
      [acrossLine('abcde')],
      [createBlocker('box', -32.5, 13.5)],
      edgeRouting,
    )

    expect(resolved!.titles[0]?.words).toBe('abcde')
    expect(getCenter(resolved!).x - 25).toBeGreaterThanOrEqual(13.5)
    expect(getCenter(resolved!).x - 25).toBeLessThan(13.5 + A_STEP + 0.01)
    expect(resolved!.titles[0]!.wordsAt).toBeCloseTo(0.6025, 3)
  })

  it('remains at the clear place nearest the middle, to the left of a box', () => {
    const [resolved] = separateTitles(
      [acrossLine('abcde')],
      [createBlocker('box', -13.5, 32.5)],
      edgeRouting,
    )

    expect(resolved!.titles[0]?.words).toBe('abcde')
    expect(getCenter(resolved!).x + 25).toBeLessThanOrEqual(-13.5)
    expect(getCenter(resolved!).x + 25).toBeGreaterThan(-13.5 - A_STEP - 0.01)
    expect(resolved!.titles[0]!.wordsAt).toBeCloseTo(0.4025, 3)
  })

  it('counts the fraction a title is read from along the line it is read on', () => {
    // The same line taken right to left, so the title is read from its far end.
    const backwards = createLine(ACROSS.to, ACROSS.from, 'abcde')
    const [resolved] = separateTitles([backwards], [createBlocker('box', -32.5, 13.5)], edgeRouting)

    expect(resolved!.titles[0]!.heading).toBe('against')
    expect(resolved!.titles[0]!.wordsAt).toBeCloseTo(0.5975, 3)
    expect(getCenter(resolved!).x - 25).toBeGreaterThanOrEqual(13.5)
  })

  it('starts a title at the near end of the only clear place that is to the right of the middle', () => {
    // A box over the middle and the whole left side of the line.
    const [resolved] = separateTitles(
      [acrossLine('abcde')],
      [createBlocker('box', -204, 53)],
      edgeRouting,
    )

    expect(resolved!.titles[0]?.words).toBe('abcde')
    expect(getCenter(resolved!).x - 25).toBeGreaterThanOrEqual(53)
    expect(getCenter(resolved!).x - 25).toBeLessThan(53 + A_STEP + 0.01)
  })

  it('clears the titles already set by what they take up, and no more', () => {
    // Two lines a line of type apart: the first keeps the middle, and the
    // second goes round it by the nearer side.
    const upper = acrossLine('abcde')
    const lower = createLine({ x: -192, y: 10 }, { x: 208, y: 10 }, 'abcde')
    const [first, second] = separateTitles([upper, lower], [], edgeRouting)

    expect(first!.titles[0]!.wordsAt).toBe(MIDDLE)
    expect(getCenter(second!).x - 25).toBeGreaterThanOrEqual(25)
    expect(getCenter(second!).x - 25).toBeLessThan(25 + A_STEP + 0.01)
  })

  it('sets the line with the least room first, whatever order they come in', () => {
    // The roomy line comes first, and its title would take the middle that the
    // tight line, barely longer than its words, has no other place for.
    const roomy = createLine({ x: -45, y: 0 }, { x: 45, y: 0 }, 'abcde')
    const tight = createLine({ x: -40, y: 10 }, { x: 40, y: 10 }, 'abcdefg')
    const [settledRoomy, settledTight] = separateTitles([roomy, tight], [], edgeRouting)

    expect(settledTight!.titles[0]?.words).toBe('abcdefg')
    expect(settledTight!.titles[0]!.wordsAt).toBeCloseTo(MIDDLE, 5)
    expect(settledRoomy!.titles[0]?.words).toBeUndefined()
  })

  it('leaves a title where it was put when its words measure nothing', () => {
    const placedLine = createLine(ACROSS.from, ACROSS.to, 'abcde', { wordsAt: 0.3 })
    const [resolved] = separateTitles(
      [placedLine],
      [],
      createRouting(DEFAULT_OPTIONS, () => 0, labelType),
    )

    expect(resolved!.titles[0]?.words).toBe('abcde')
    expect(resolved!.titles[0]!.wordsAt).toBe(0.3)
  })

  it('leaves a title where it was put when its line has no length', () => {
    const point = { x: 0, y: 0 }
    const placedLine = createLine(point, point, 'abcde', { wordsAt: 0.3 })
    const [resolved] = separateTitles([placedLine], [], edgeRouting)

    expect(resolved!.titles[0]?.words).toBe('abcde')
    expect(resolved!.titles[0]!.wordsAt).toBe(0.3)
  })

  it('cuts a title to the longest clear place when no place holds the whole of it', () => {
    // Three places of about 60, 105 and 80 between boxes, for words 120 long.
    const boxes = [
      createBlocker('one', -140, -122),
      createBlocker('two', -17, 0),
      createBlocker('three', 80, 200),
    ]
    const [resolved] = separateTitles([acrossLine('abcdefghijkl')], boxes, edgeRouting)

    expect(resolved!.titles[0]?.words).toBe('abcdefghi…')
    expect(getCenter(resolved!).x).toBeGreaterThan(-122)
    expect(getCenter(resolved!).x).toBeLessThan(-17)
  })

  it('keeps the room an arrowhead takes at the end of the line', () => {
    // Clear only from 136 to the end, which an arrowhead takes 20 of.
    const arrow = createLine(ACROSS.from, ACROSS.to, 'abcdefgh', { arrows: ['to'] })
    const [resolved] = separateTitles([arrow], [createBlocker('box', -204, 134)], edgeRouting)

    expect(resolved!.titles[0]?.words).toBe('abc…')
    expect(getCenter(resolved!).x + TEN(resolved!.titles[0]!.words) / 2).toBeLessThanOrEqual(180)
  })

  it('writes no title on a line that an arrowhead takes the whole of', () => {
    const short = createLine({ x: 0, y: 0 }, { x: 30, y: 0 }, 'a', { arrows: ['to'] })
    const [resolved] = separateTitles([short], [], edgeRouting)

    expect(resolved!.titles[0]?.words).toBeUndefined()
  })

  it('keeps a title clear of the corner of a slanting title that a box around it would take', () => {
    // The first title slants across the page; the second line runs below it,
    // under the empty corner of the box around the first.
    const slanting = createLine({ x: -100, y: -100 }, { x: 100, y: 100 }, 'abcde')
    const below = createLine({ x: -200, y: 28 }, { x: 200, y: 28 }, 'abcde')
    const [, second] = separateTitles([slanting, below], [], edgeRouting)

    expect(getCenter(second!).x).toBeGreaterThan(-30)
    expect(getCenter(second!).x).toBeLessThan(-10)
  })
})

/**
 * A line with one title keeps it in the middle; a line with two keeps one near
 * each end, whatever arrowheads the line has.
 */
describe('the titles of one line', () => {
  const TEN = perLetter(10)
  const edgeRouting = createRouting(DEFAULT_OPTIONS, TEN, labelType)
  const FROM = { x: -200, y: 0 }
  const TO = { x: 200, y: 0 }

  /** A straight line from left to right, with a title for each label given, at its asked place. */
  const createLine = (labels: readonly EdgeLabel[], arrows: readonly EdgeArrow[]): PlacedEdge => {
    const curve = {
      fromPoint: FROM,
      control1: { x: -200 / 3, y: 0 },
      control2: { x: 200 / 3, y: 0 },
      toPoint: TO,
    }
    const places = getLabelPlaces(labels)
    return {
      from: 'harbour',
      to: 'lighthouse',
      ...curve,
      labels,
      arrows,
      opacity: 1,
      titles: labels.map((label, index) =>
        createTitle(curve, places[index]!.position, label.text, setting(places[index]!)),
      ),
    }
  }

  /** The x of the middle of each title, on a line that runs left to right. */
  const getCenters = (edge: PlacedEdge) =>
    edge.titles.map(
      (title) =>
        FROM.x +
        (TO.x - FROM.x) * (title.heading === 'against' ? 1 - title.wordsAt : title.wordsAt),
    )

  it('keeps one title in the middle when the line is clear, whatever arrowhead it has', () => {
    for (const arrows of [[], ['to'], ['from'], ['to', 'from']] as const) {
      const [resolved] = separateTitles([createLine([{ text: 'abcde' }], arrows)], [], edgeRouting)

      expect(getCenters(resolved!)).toEqual([0])
    }
  })

  it('moves to the clear place nearest the middle, short of the arrowhead, when its words are long', () => {
    // 160 wide in a line of 400 with 20 kept at each end for a head: it fits only
    // between them, and the middle is such a place.
    const [resolved] = separateTitles(
      [createLine([{ text: 'abcdefghijklmnop' }], ['to', 'from'])],
      [],
      edgeRouting,
    )
    const [center] = getCenters(resolved!)

    expect(resolved!.titles[0]!.words).toBe('abcdefghijklmnop')
    expect(center).toBeCloseTo(0)
  })

  it('stays on its half of the line when a box covers the middle', () => {
    const box: PlacedNode = {
      id: 'buoy',
      title: 'buoy',
      relation: 'sibling',
      x: 0,
      y: 0,
      width: 30,
      height: 20,
      order: 0,
      opacity: 1,
    }
    const [resolved] = separateTitles([createLine([{ text: 'abcde' }], [])], [box], edgeRouting)
    const center = getCenters(resolved!)[0]!

    expect(Math.abs(center)).toBeGreaterThan(15 + 25)
  })

  it('keeps both titles of a line, the first near the start and the second near the end', () => {
    const [resolved] = separateTitles(
      [createLine([{ text: 'guides' }, { text: 'shelters' }], ['to', 'from'])],
      [],
      edgeRouting,
    )

    expect(resolved!.titles.map((title) => title.words)).toEqual(['guides', 'shelters'])
    const [first, second] = getCenters(resolved!)
    expect(first).toBeLessThan(FROM.x + 400 / 3)
    expect(second).toBeGreaterThan(TO.x - 400 / 3)
  })

  it('never sets a title past the middle on the side of the other when a box covers its place', () => {
    // The box covers the whole first half of the line and a little more.
    const box: PlacedNode = {
      id: 'buoy',
      title: 'buoy',
      relation: 'sibling',
      x: -70,
      y: 0,
      width: 260,
      height: 20,
      order: 0,
      opacity: 1,
    }
    const [resolved] = separateTitles(
      [createLine([{ text: 'guides' }, { text: 'shelters' }], [])],
      [box],
      edgeRouting,
    )
    const centers = new Map(
      resolved!.titles.map((title, index) => [title.words, getCenters(resolved!)[index]!]),
    )

    expect(centers.get('guides') ?? -1).toBeLessThanOrEqual(0)
    expect(centers.get('shelters') ?? 1).toBeGreaterThan(0)
  })

  it('keeps the boxes of the two titles apart, at the width of a realistic label', () => {
    const [resolved] = separateTitles(
      [createLine([{ text: 'is a part of' }, { text: 'consists of' }], [])],
      [],
      edgeRouting,
    )
    const [first, second] = getCenters(resolved!)
    const half = (words: string) => TEN(words) / 2

    expect(resolved!.titles).toHaveLength(2)
    expect(second! - first!).toBeGreaterThanOrEqual(half('is a part of') + half('consists of'))
  })
})
