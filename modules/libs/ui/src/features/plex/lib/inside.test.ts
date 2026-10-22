/**
 * The headings of a node: which of them are drawn, how far each is set in, and
 * where each stands partway through the opening.
 */
import { describe, expect, it } from 'vitest'
import { easeOut } from './arrange'
import {
  getNodeHeadings as baseGetNodeHeadings,
  type HeadingsDeps,
  type PlexHeading,
} from './inside'
import { getFurthest, getOpenHeadings, scrollBy } from './open'
import type { PlacedNode } from './node'

const NODE: PlacedNode = {
  id: 'a',
  title: 'A node',
  relation: 'focus',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

/** How many headings stand in the window at once, as the options ask for. */
const MOST = 6

const SIZES = { headingHeight: 20, headingIndent: 10, maxHeadings: MOST }

/** A window with room to spare, and a measurer of ten pixels the letter. */
const DEPS: HeadingsDeps = {
  measure: (text: string) => 10 * text.length,
  viewport: { width: 1000, height: 600 },
  margin: 20,
}

const getNodeHeadings = (
  node: PlacedNode,
  items: readonly PlexHeading[],
  options = SIZES,
  deps = DEPS,
) => baseGetNodeHeadings(node, items, { options, deps })

/** Headings all at one level, named by their number. */
const headings = (count: number, level = 1): PlexHeading[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `${index}`,
    text: `Heading ${index}`,
    level,
  }))

const suspended = (items: readonly PlexHeading[], deps = DEPS) =>
  getNodeHeadings(NODE, items, SIZES, deps)

describe('what a node hangs', () => {
  it('is nothing at all for a node with no headings', () => {
    expect(suspended([])).toBeNull()
  })

  it('arrives out from under the box, which is the node’s own bottom edge', () => {
    expect(suspended(headings(1))?.top).toBe(NODE.height / 2)
  })

  it('holds them in the order they were given', () => {
    const target = suspended(headings(3))?.headings ?? []
    expect(target.map((heading) => heading.text)).toStrictEqual([
      'Heading 0',
      'Heading 1',
      'Heading 2',
    ])
  })

  it('rests each heading a heading’s height below the one above it', () => {
    const target = suspended(headings(3))?.headings ?? []
    expect(target.map((heading) => heading.at)).toStrictEqual([0, 20, 40])
  })

  it('remains as deep as the headings it holds, and the ground it keeps clear', () => {
    const resolved = suspended(headings(3))!
    expect(resolved.height).toBe(3 * SIZES.headingHeight + 2 * resolved.pad)
    expect(resolved.pad).toBeGreaterThan(0)
  })
})

describe('more headings than the window holds', () => {
  it('remains the ceiling of them in it, and keeps every one of them', () => {
    const resolved = suspended(headings(MOST + 4))!
    expect(resolved.visibleCount).toBe(MOST)
    expect(resolved.headings).toHaveLength(MOST + 4)
    expect(getFurthest(resolved)).toBe(4)
  })

  it('has nowhere to scroll where every one of them remains at once', () => {
    const resolved = suspended(headings(MOST))!
    expect(resolved.visibleCount).toBe(MOST)
    expect(getFurthest(resolved)).toBe(0)
  })

  it('remains as deep as the window, not as deep as it holds', () => {
    const resolved = suspended(headings(MOST + 4))!
    expect(resolved.height).toBe(MOST * SIZES.headingHeight + 2 * resolved.pad)
  })

  it('remains as many as the options ask for', () => {
    const resolved = getNodeHeadings(NODE, headings(MOST + 4), { ...SIZES, maxHeadings: 3 }, DEPS)!
    expect(resolved.visibleCount).toBe(3)
    expect(getFurthest(resolved)).toBe(MOST + 1)
  })
})

describe('scrolling the window over the headings', () => {
  const many = () => suspended(headings(MOST + 3))!

  it('opens on the first of them, with more below and none above', () => {
    const visibleHeadings = getOpenHeadings(many(), 1)!
    expect(visibleHeadings.headings[0]!.text).toBe('Heading 0')
    expect(visibleHeadings.isAbove).toBe(false)
    expect(visibleHeadings.isBelow).toBe(true)
  })

  it('moves by whole headings, so none is ever half on the ground', () => {
    const visibleHeadings = getOpenHeadings(many(), 1, 2)!
    expect(visibleHeadings.headings[0]!.text).toBe('Heading 2')
    expect(visibleHeadings.headings.map((heading) => heading.at)).toStrictEqual(
      visibleHeadings.headings.map((_, index) => index * SIZES.headingHeight),
    )
  })

  it('says there is more above it once it has been scrolled', () => {
    expect(getOpenHeadings(many(), 1, 1)!.isAbove).toBe(true)
  })

  it('scrolls no further than the last of them', () => {
    const resolved = many()
    const visibleHeadings = getOpenHeadings(resolved, 1, 99)!
    expect(visibleHeadings.headings.at(-1)!.text).toBe(`Heading ${resolved.headings.length - 1}`)
    expect(visibleHeadings.isBelow).toBe(false)
  })

  it('scrolls no further back than the first of them', () => {
    expect(getOpenHeadings(many(), 1, -5)!.headings[0]!.text).toBe('Heading 0')
  })

  it('remains the window full however far it is scrolled', () => {
    const resolved = many()
    for (const scrollOffset of [0, 1, 2, 3, 99]) {
      expect(getOpenHeadings(resolved, 1, scrollOffset)!.headings).toHaveLength(resolved.visibleCount)
    }
  })
})

describe('how far in a heading is set', () => {
  const getHeadingIndents = (levels: readonly number[]) =>
    (
      suspended(levels.map((level, index) => ({ id: `${index}`, text: `Heading ${index}`, level })))?.headings ??
      []
    ).map((heading) => heading.indent)

  it('is by where its level remains among the others, not by the number it carries', () => {
    // Headings beginning at the second level are set from the edge as headings
    // beginning at the first.
    expect(getHeadingIndents([2, 3, 3])).toStrictEqual([0, 10, 10])
  })

  it('counts a level that was skipped as one step, not two', () => {
    expect(getHeadingIndents([1, 3])).toStrictEqual([0, 10])
  })

  it('ranks the levels as numbers, whatever order they were given in', () => {
    expect(getHeadingIndents([10, 2, 3])).toStrictEqual([20, 0, 10])
    expect(getHeadingIndents([3, 2, 1])).toStrictEqual([20, 10, 0])
  })

  it('goes no further in however deep the nesting runs', () => {
    expect(getHeadingIndents([1, 2, 3, 4, 5, 6])).toStrictEqual([0, 10, 20, 30, 30, 30])
  })
})

describe('the opening', () => {
  const openTo = (open: number, count = 3) => getOpenHeadings(suspended(headings(count))!, open)

  it('draws nothing while it is shut', () => {
    expect(openTo(0)).toBeNull()
  })

  it('has its ground barely up before any of them has risen', () => {
    expect(openTo(0.0001)?.opacity).toBeLessThan(0.01)
  })

  it('brings the ground up at the depth it keeps', () => {
    const resolved = suspended(headings(3))!
    for (const open of [0.2, 0.5, 1]) {
      expect(getOpenHeadings(resolved, open)!.height).toBe(resolved.height)
    }
    expect(getOpenHeadings(resolved, 1)!.opacity).toBe(1)
  })

  it('has the first heading further along than the last', () => {
    const renderedHeadings = openTo(0.5)?.headings ?? []
    expect(renderedHeadings[0]!.opacity).toBeGreaterThan(renderedHeadings.at(-1)!.opacity)
  })

  it('rises onto its place from below, never from above it', () => {
    const resolved = suspended(headings(3))!
    getOpenHeadings(resolved, 0.5)!.headings.forEach((heading, index) => {
      const rests = resolved.headings[index]!.at
      expect(heading.y).toBeGreaterThanOrEqual(rests)
      expect(heading.y).toBeLessThanOrEqual(rests + resolved.headingHeight)
    })
  })

  it('keeps every heading on the ground, at every moment of the opening', () => {
    // A heading drawn past either edge is a heading standing on nothing, which is
    // what the ground is there to stop.
    const resolved = suspended(headings(5))!
    for (const open of [0.05, 0.2, 0.4, 0.6, 0.8, 1]) {
      for (const heading of getOpenHeadings(resolved, open)!.headings) {
        expect(heading.y).toBeGreaterThanOrEqual(0)
        expect(heading.y + resolved.headingHeight).toBeLessThanOrEqual(
          resolved.height - 2 * resolved.pad,
        )
      }
    }
  })

  it('rests every heading where it hangs once it is all the way open', () => {
    const resolved = suspended(headings(3))!
    const renderedHeadings = getOpenHeadings(resolved, 1)?.headings ?? []
    expect(renderedHeadings.map((heading) => heading.y)).toStrictEqual(
      resolved.headings.map((heading) => heading.at),
    )
    expect(renderedHeadings.every((heading) => heading.opacity === 1)).toBe(true)
  })

  it('brings the ground up before the headings have finished rising', () => {
    const resolved = suspended(headings(3))!
    const half = getOpenHeadings(resolved, 0.5)!
    expect(half.opacity).toBeGreaterThan(half.headings.at(-1)!.opacity)
  })

  it('carries the words and the indent through untouched', () => {
    const heading = openTo(1)?.headings[1]
    expect(heading?.text).toBe('Heading 1')
    expect(heading?.id).toBe('1')
    expect(heading?.indent).toBe(0)
  })
})

describe('how wide the headings are drawn', () => {
  const measureHeadings = (items: readonly PlexHeading[], deps = DEPS) => suspended(items, deps)!

  it('is the room the longest of them asks for, and the ground alongside it', () => {
    const longest = 'A good deal longer than that'
    const target = [
      { id: '0', text: 'Short', level: 1 },
      { id: '1', text: longest, level: 1 },
    ]
    const resolved = measureHeadings(target)
    expect(resolved.width).toBe(10 * longest.length + 2 * resolved.pad)
  })

  it('is never narrower than the node’s own box', () => {
    expect(measureHeadings([{ id: '0', text: 'Wee', level: 1 }]).width).toBe(NODE.width)
  })

  it('counts how far a heading is set in as room it needs', () => {
    const words = 'A'.repeat(20)
    const flat = [{ id: '0', text: words, level: 1 }]
    const nested = [...flat, { id: '1', text: words, level: 2 }]

    expect(measureHeadings(nested).width - measureHeadings(flat).width).toBe(SIZES.headingIndent)
  })

  it('is the node’s own box where there is nothing to measure text with', () => {
    const bareDeps = { viewport: DEPS.viewport, margin: DEPS.margin }
    expect(measureHeadings(headings(3), bareDeps).width).toBe(NODE.width)
  })

  it('is retained inside the window, however long the words run', () => {
    const long = [{ id: '0', text: 'x'.repeat(400), level: 1 }]
    expect(measureHeadings(long).width).toBe(DEPS.viewport.width - 2 * DEPS.margin)
  })

  it('grows about the node’s own middle where the window has room', () => {
    expect(measureHeadings(headings(2)).offset).toBe(0)
  })

  it('slides back inside the window where the middle leaves no room', () => {
    const nearDistances = { ...NODE, x: 460 }
    const target = [{ id: '0', text: 'A heading of some length', level: 1 }]
    const resolved = getNodeHeadings(nearDistances, target, SIZES, DEPS)!
    expect(nearDistances.x + resolved.offset + resolved.width / 2).toBeLessThanOrEqual(
      DEPS.viewport.width / 2 - DEPS.margin,
    )
  })
})

describe('where the headings slide to at the edges of the window', () => {
  const target = [{ id: '0', text: 'A heading of some length', level: 1 }]
  const edge = DEPS.viewport.width / 2 - DEPS.margin

  it('ends flush with the margin on the right', () => {
    const nearDistances = { ...NODE, x: 460 }
    const resolved = getNodeHeadings(nearDistances, target, SIZES, DEPS)!
    expect(nearDistances.x + resolved.offset + resolved.width / 2).toBe(edge)
  })

  it('begins flush with the margin on the left', () => {
    const nearDistances = { ...NODE, x: -460 }
    const resolved = getNodeHeadings(nearDistances, target, SIZES, DEPS)!
    expect(nearDistances.x + resolved.offset - resolved.width / 2).toBe(-edge)
  })
})

describe('a node with little room under it', () => {
  /** The same node, dropped so much of the window is left beneath it. */
  const low = (roomOffset: number): PlacedNode => ({
    ...NODE,
    y: DEPS.viewport.height / 2 - DEPS.margin - NODE.height / 2 - roomOffset,
  })

  const hangWithRoom = (roomOffset: number, items = headings(MOST + 3)) =>
    getNodeHeadings(low(roomOffset), items, SIZES, DEPS)

  it('hangs nothing where there is depth for not one heading', () => {
    expect(hangWithRoom(SIZES.headingHeight)).toBeNull()
  })

  it('remains in its window only what the depth left under it holds', () => {
    const resolved = hangWithRoom(3 * SIZES.headingHeight + 10)!
    expect(resolved.visibleCount).toBe(3)
    expect(getFurthest(resolved)).toBe(MOST)
  })

  it('hangs one heading where the depth holds exactly one with its ground', () => {
    const resolved = hangWithRoom(30)!
    expect(resolved.visibleCount).toBe(1)
    expect(resolved.height).toBe(SIZES.headingHeight + 2 * resolved.pad)
  })

  it('hangs nothing where the depth is one short of that', () => {
    expect(hangWithRoom(29)).toBeNull()
  })

  it('keeps every heading it does hang inside the window', () => {
    for (const roomOffset of [60, 90, 140, 200, 400]) {
      const resolved = hangWithRoom(roomOffset)
      if (!resolved) continue
      const bottom = low(roomOffset).y + resolved.top + resolved.height
      expect(bottom).toBeLessThanOrEqual(DEPS.viewport.height / 2 - DEPS.margin)
    }
  })

  it('remains them all at once where the depth holds every one of them', () => {
    const resolved = getNodeHeadings(low(400), headings(3), SIZES, DEPS)!
    expect(resolved.visibleCount).toBe(3)
    expect(getFurthest(resolved)).toBe(0)
  })
})

describe('what a wheel scrolls', () => {
  const hangMany = () => suspended(headings(20))!
  /** A wheel said in pixels, which is what a hand on a trackpad gives. */
  const pixels = (delta: number) => ({ delta, mode: 0 })

  it('is nothing at all until the pixels come to a whole heading', () => {
    const wheel = scrollBy(hangMany(), pixels(SIZES.headingHeight - 1), 0)
    expect(wheel.by).toBe(0)
    expect(wheel.left).toBe(SIZES.headingHeight - 1)
  })

  it('carries what was left over into the next one', () => {
    const first = scrollBy(hangMany(), pixels(12), 0)
    const next = scrollBy(hangMany(), pixels(12), first.left)
    expect(first.by).toBe(0)
    expect(next.by).toBe(1)
    expect(next.left).toBe(4)
  })

  it('is one heading for a flick of a hand, not one for every event it sends', () => {
    // A trackpad gives a few pixels at a time, dozens of times a flick.
    let carried = 0
    let scrollOffset = 0
    for (let step = 0; step < 8; step++) {
      const wheel = scrollBy(hangMany(), pixels(3), carried)
      carried = wheel.left
      scrollOffset += wheel.by
    }
    expect(scrollOffset).toBe(1)
  })

  it('reads a wheel returned in lines as one heading the line', () => {
    expect(scrollBy(hangMany(), { delta: 2, mode: 1 }, 0).by).toBe(2)
  })

  it('reads a wheel returned in windows as the whole window', () => {
    const target = hangMany()
    expect(scrollBy(target, { delta: 1, mode: 2 }, 0).by).toBe(target.visibleCount)
  })

  it('scrolls back the way it came, and carries the leftover the same way', () => {
    const back = scrollBy(hangMany(), pixels(-SIZES.headingHeight - 6), 0)
    expect(back.by).toBe(-1)
    expect(back.left).toBe(-6)
  })
})

describe('however many headings stand at once', () => {
  /** A node with depth under it for as many headings as the ceiling allows. */
  const hangWithCeiling = (most: number) =>
    getNodeHeadings(NODE, headings(most + 4), { ...SIZES, maxHeadings: most }, DEPS)!

  it('every one of them is up by the time it is all the way open', () => {
    // A lead that outran the opening left the last of them at nothing at all,
    // on a ground drawn deep enough to hold them.
    for (const ceiling of [1, 2, 6, 9, 12, 20]) {
      const resolved = hangWithCeiling(ceiling)
      const renderedHeadings = getOpenHeadings(resolved, 1)!.headings
      expect(renderedHeadings).toHaveLength(resolved.visibleCount)
      for (const heading of renderedHeadings) expect(heading.opacity).toBe(1)
    }
  })

  it('the first of them is still ahead of the last partway through', () => {
    for (const ceiling of [2, 6, 12, 20]) {
      const renderedHeadings = getOpenHeadings(hangWithCeiling(ceiling), 0.5)!.headings
      expect(renderedHeadings[0]!.opacity).toBeGreaterThan(renderedHeadings.at(-1)!.opacity)
    }
  })

  it('one alone opens with the whole of the opening to itself', () => {
    expect(getOpenHeadings(hangWithCeiling(1), 0.5)!.headings[0]!.opacity).toBe(easeOut(0.5))
  })
})

describe('when the leads set off', () => {
  /** The opacity of the last heading shown, at a moment of the opening. */
  const getLast = (ceiling: number, open: number) => {
    const resolved = getNodeHeadings(
      NODE,
      headings(ceiling),
      { ...SIZES, maxHeadings: ceiling },
      DEPS,
    )!
    return getOpenHeadings(resolved, open)!.headings.at(-1)!.opacity
  }

  it('has the last of a few headings still down while two leads of the opening are taken', () => {
    expect(getLast(3, 0.23)).toBe(0)
    expect(getLast(3, 0.26)).toBeGreaterThan(0)
  })

  it('has the leads of many headings take the same share between them', () => {
    for (const ceiling of [8, 20]) {
      expect(getLast(ceiling, 0.59)).toBe(0)
      expect(getLast(ceiling, 0.61)).toBeGreaterThan(0)
    }
  })
})

describe('where a heading sets off from', () => {
  it('is a fraction of its own height below its place', () => {
    const resolved = suspended(headings(3))!
    const first = getOpenHeadings(resolved, 0.3)!.headings[0]!
    expect(first.opacity).toBeGreaterThan(0)
    expect(first.opacity).toBeLessThan(1)
    expect(first.y).toBeCloseTo((1 - first.opacity) * 0.7 * resolved.headingHeight, 6)
  })
})

describe('the arrows at the edges of the ground', () => {
  const WIDE = 4
  const DEEP = 2.5
  const many = () => suspended(headings(MOST + 3))!

  it('points down from the bottom of the ground where there is more below', () => {
    const resolved = many()
    const { arrows } = getOpenHeadings(resolved, 1, 0)!
    const y = resolved.top + resolved.height - resolved.pad / 2
    expect(arrows).toStrictEqual([
      {
        at: 'below',
        points: [
          { x: -WIDE, y: y - DEEP },
          { x: 0, y: y + DEEP },
          { x: WIDE, y: y - DEEP },
        ],
      },
    ])
  })

  it('points up from the top of the ground where there is more above', () => {
    const resolved = many()
    const { arrows } = getOpenHeadings(resolved, 1, getFurthest(resolved))!
    const y = resolved.top + resolved.pad / 2
    expect(arrows).toStrictEqual([
      {
        at: 'above',
        points: [
          { x: -WIDE, y: y + DEEP },
          { x: 0, y: y - DEEP },
          { x: WIDE, y: y + DEEP },
        ],
      },
    ])
  })

  it('draws both, the one above first, where the window is scrolled between', () => {
    const { arrows } = getOpenHeadings(many(), 1, 1)!
    expect(arrows.map((arrow) => arrow.at)).toStrictEqual(['above', 'below'])
  })

  it('draws none where every heading remains at once', () => {
    expect(getOpenHeadings(suspended(headings(3))!, 1)!.arrows).toStrictEqual([])
  })

  it('is drawn about the middle of what is hung, not of the node', () => {
    const nearDistances = { ...NODE, x: 460 }
    const heading = { id: 'h', text: 'A heading of some length', level: 1 }
    const resolved = getNodeHeadings(nearDistances, [heading, ...headings(MOST + 3)], SIZES, DEPS)!
    const [arrow] = getOpenHeadings(resolved, 1, 0)!.arrows
    expect(arrow!.points.map((point) => point.x)).toStrictEqual([
      resolved.offset - WIDE,
      resolved.offset,
      resolved.offset + WIDE,
    ])
  })
})
