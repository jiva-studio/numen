/**
 * The parts of a node: which of them are drawn, how far each is set in, and
 * where each stands partway through the opening.
 */
import { describe, expect, it } from 'vitest'
import { easeOut } from './arrange'
import { hangParts, type PartsDeps, type PlexPart } from './inside'
import { furthest, getOpenParts, woundBy } from './open'
import type { PlacedNode } from './node'

const NODE: PlacedNode = {
  id: 'a',
  title: 'A node',
  seat: 'focus',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

/** How many parts stand in the window at once, as the options ask for. */
const MOST = 6

const SIZES = { partHeight: 20, partIndent: 10, maxParts: MOST }

/** A window with room to spare, and a measurer of ten pixels the letter. */
const DEPS: PartsDeps = {
  measure: (text: string) => 10 * text.length,
  viewport: { width: 1000, height: 600 },
  margin: 20,
}

/** Parts all at one level, named by their number. */
const parts = (count: number, level = 1): PlexPart[] =>
  Array.from({ length: count }, (_, at) => ({
    id: `${at}`,
    text: `Part ${at}`,
    level,
  }))

const hung = (held: readonly PlexPart[], deps = DEPS) => hangParts(NODE, held, SIZES, deps)

describe('what a node hangs', () => {
  it('is nothing at all for a node with no parts', () => {
    expect(hung([])).toBeNull()
  })

  it('comes out from under the box, which is the node’s own bottom edge', () => {
    expect(hung(parts(1))?.top).toBe(NODE.height / 2)
  })

  it('holds them in the order they were given', () => {
    const held = hung(parts(3))?.parts ?? []
    expect(held.map((part) => part.text)).toStrictEqual(['Part 0', 'Part 1', 'Part 2'])
  })

  it('rests each part a part’s height below the one above it', () => {
    const held = hung(parts(3))?.parts ?? []
    expect(held.map((part) => part.at)).toStrictEqual([0, 20, 40])
  })

  it('stands as deep as the parts it holds, and the ground it keeps clear', () => {
    const settled = hung(parts(3))!
    expect(settled.height).toBe(3 * SIZES.partHeight + 2 * settled.pad)
    expect(settled.pad).toBeGreaterThan(0)
  })
})

describe('more parts than the window holds', () => {
  it('stands the ceiling of them in it, and keeps every one of them', () => {
    const settled = hung(parts(MOST + 4))!
    expect(settled.shown).toBe(MOST)
    expect(settled.parts).toHaveLength(MOST + 4)
    expect(furthest(settled)).toBe(4)
  })

  it('has nowhere to wind where every one of them stands at once', () => {
    const settled = hung(parts(MOST))!
    expect(settled.shown).toBe(MOST)
    expect(furthest(settled)).toBe(0)
  })

  it('stands as deep as the window, not as deep as it holds', () => {
    const settled = hung(parts(MOST + 4))!
    expect(settled.height).toBe(MOST * SIZES.partHeight + 2 * settled.pad)
  })

  it('stands as many as the options ask for', () => {
    const settled = hangParts(NODE, parts(MOST + 4), { ...SIZES, maxParts: 3 }, DEPS)!
    expect(settled.shown).toBe(3)
    expect(furthest(settled)).toBe(MOST + 1)
  })
})

describe('winding the window over the parts', () => {
  const many = () => hung(parts(MOST + 3))!

  it('opens on the first of them, with more below and none above', () => {
    const shown = getOpenParts(many(), 1)!
    expect(shown.parts[0]!.text).toBe('Part 0')
    expect(shown.above).toBe(false)
    expect(shown.below).toBe(true)
  })

  it('moves by whole parts, so none is ever half on the ground', () => {
    const shown = getOpenParts(many(), 1, 2)!
    expect(shown.parts[0]!.text).toBe('Part 2')
    expect(shown.parts.map((part) => part.at)).toStrictEqual(
      shown.parts.map((_, at) => at * SIZES.partHeight),
    )
  })

  it('says there is more above it once it has been wound', () => {
    expect(getOpenParts(many(), 1, 1)!.above).toBe(true)
  })

  it('winds no further than the last of them', () => {
    const settled = many()
    const shown = getOpenParts(settled, 1, 99)!
    expect(shown.parts.at(-1)!.text).toBe(`Part ${settled.parts.length - 1}`)
    expect(shown.below).toBe(false)
  })

  it('winds no further back than the first of them', () => {
    expect(getOpenParts(many(), 1, -5)!.parts[0]!.text).toBe('Part 0')
  })

  it('stands the window full however far it is wound', () => {
    const settled = many()
    for (const wound of [0, 1, 2, 3, 99]) {
      expect(getOpenParts(settled, 1, wound)!.parts).toHaveLength(settled.shown)
    }
  })
})

describe('how far in a part is set', () => {
  const setIn = (levels: readonly number[]) =>
    (
      hung(
        levels.map((level, at) => ({ id: `${at}`, text: `Part ${at}`, level })),
      )?.parts ?? []
    ).map((part) => part.indent)

  it('is by where its level stands among the others, not by the number it carries', () => {
    // Parts beginning at the second level are set from the edge as parts
    // beginning at the first.
    expect(setIn([2, 3, 3])).toStrictEqual([0, 10, 10])
  })

  it('counts a level that was skipped as one step, not two', () => {
    expect(setIn([1, 3])).toStrictEqual([0, 10])
  })

  it('goes no further in however deep the nesting runs', () => {
    expect(setIn([1, 2, 3, 4, 5, 6])).toStrictEqual([0, 10, 20, 30, 30, 30])
  })
})

describe('the opening', () => {
  const openTo = (open: number, count = 3) => getOpenParts(hung(parts(count))!, open)

  it('draws nothing while it is shut', () => {
    expect(openTo(0)).toBeNull()
  })

  it('has its ground barely up before any of them has risen', () => {
    expect(openTo(0.0001)?.opacity).toBeLessThan(0.01)
  })

  it('brings the ground up at the depth it keeps', () => {
    const settled = hung(parts(3))!
    for (const open of [0.2, 0.5, 1]) {
      expect(getOpenParts(settled, open)!.height).toBe(settled.height)
    }
    expect(getOpenParts(settled, 1)!.opacity).toBe(1)
  })

  it('has the first part further along than the last', () => {
    const drawn = openTo(0.5)?.parts ?? []
    expect(drawn[0]!.opacity).toBeGreaterThan(drawn.at(-1)!.opacity)
  })

  it('rises onto its place from below, never from above it', () => {
    const settled = hung(parts(3))!
    getOpenParts(settled, 0.5)!.parts.forEach((part, at) => {
      const rests = settled.parts[at]!.at
      expect(part.y).toBeGreaterThanOrEqual(rests)
      expect(part.y).toBeLessThanOrEqual(rests + settled.partHeight)
    })
  })

  it('keeps every part on the ground, at every moment of the opening', () => {
    // A part drawn past either edge is a part standing on nothing, which is
    // what the ground is there to stop.
    const settled = hung(parts(5))!
    for (const open of [0.05, 0.2, 0.4, 0.6, 0.8, 1]) {
      for (const part of getOpenParts(settled, open)!.parts) {
        expect(part.y).toBeGreaterThanOrEqual(0)
        expect(part.y + settled.partHeight).toBeLessThanOrEqual(
          settled.height - 2 * settled.pad,
        )
      }
    }
  })

  it('rests every part where it hangs once it is all the way open', () => {
    const settled = hung(parts(3))!
    const drawn = getOpenParts(settled, 1)?.parts ?? []
    expect(drawn.map((part) => part.y)).toStrictEqual(settled.parts.map((part) => part.at))
    expect(drawn.every((part) => part.opacity === 1)).toBe(true)
  })

  it('brings the ground up before the parts have finished rising', () => {
    const settled = hung(parts(3))!
    const half = getOpenParts(settled, 0.5)!
    expect(half.opacity).toBeGreaterThan(half.parts.at(-1)!.opacity)
  })

  it('carries the words and the indent through untouched', () => {
    const part = openTo(1)?.parts[1]
    expect(part?.text).toBe('Part 1')
    expect(part?.id).toBe('1')
    expect(part?.indent).toBe(0)
  })
})

describe('how wide the parts are drawn', () => {
  const wide = (held: readonly PlexPart[], deps = DEPS) => hung(held, deps)!

  it('is the room the longest of them asks for, and the ground beside it', () => {
    const longest = 'A good deal longer than that'
    const held = [
      { id: '0', text: 'Short', level: 1 },
      { id: '1', text: longest, level: 1 },
    ]
    const settled = wide(held)
    expect(settled.width).toBe(10 * longest.length + 2 * settled.pad)
  })

  it('is never narrower than the node’s own box', () => {
    expect(wide([{ id: '0', text: 'Wee', level: 1 }]).width).toBe(NODE.width)
  })

  it('counts how far a part is set in as room it needs', () => {
    const words = 'A'.repeat(20)
    const flat = [{ id: '0', text: words, level: 1 }]
    const nested = [...flat, { id: '1', text: words, level: 2 }]

    expect(wide(nested).width - wide(flat).width).toBe(SIZES.partIndent)
  })

  it('is the node’s own box where there is nothing to measure text with', () => {
    const bare = { viewport: DEPS.viewport, margin: DEPS.margin }
    expect(wide(parts(3), bare).width).toBe(NODE.width)
  })

  it('is held inside the window, however long the words run', () => {
    const long = [{ id: '0', text: 'x'.repeat(400), level: 1 }]
    expect(wide(long).width).toBe(DEPS.viewport.width - 2 * DEPS.margin)
  })

  it('grows about the node’s own middle where the window has room', () => {
    expect(wide(parts(2)).offset).toBe(0)
  })

  it('slides back inside the window where the middle leaves no room', () => {
    const near = { ...NODE, x: 460 }
    const held = [{ id: '0', text: 'A heading of some length', level: 1 }]
    const settled = hangParts(near, held, SIZES, DEPS)!
    expect(near.x + settled.offset + settled.width / 2).toBeLessThanOrEqual(
      DEPS.viewport.width / 2 - DEPS.margin,
    )
  })
})

describe('a node with little room under it', () => {
  /** The same node, dropped so much of the window is left beneath it. */
  const low = (left: number): PlacedNode => ({
    ...NODE,
    y: DEPS.viewport.height / 2 - DEPS.margin - NODE.height / 2 - left,
  })

  const under = (left: number, held = parts(MOST + 3)) =>
    hangParts(low(left), held, SIZES, DEPS)

  it('hangs nothing where there is depth for not one part', () => {
    expect(under(SIZES.partHeight)).toBeNull()
  })

  it('stands in its window only what the depth left under it holds', () => {
    const settled = under(3 * SIZES.partHeight + 10)!
    expect(settled.shown).toBe(3)
    expect(furthest(settled)).toBe(MOST)
  })

  it('keeps every part it does hang inside the window', () => {
    for (const left of [60, 90, 140, 200, 400]) {
      const settled = under(left)
      if (!settled) continue
      const bottom = low(left).y + settled.top + settled.height
      expect(bottom).toBeLessThanOrEqual(DEPS.viewport.height / 2 - DEPS.margin)
    }
  })

  it('stands them all at once where the depth holds every one of them', () => {
    const settled = hangParts(low(400), parts(3), SIZES, DEPS)!
    expect(settled.shown).toBe(3)
    expect(furthest(settled)).toBe(0)
  })
})

describe('what a wheel winds', () => {
  const hangMany = () => hung(parts(20))!
  /** A wheel said in pixels, which is what a hand on a trackpad gives. */
  const pixels = (delta: number) => ({ delta, mode: 0 })

  it('is nothing at all until the pixels come to a whole part', () => {
    const wheel = woundBy(hangMany(), pixels(SIZES.partHeight - 1), 0)
    expect(wheel.by).toBe(0)
    expect(wheel.left).toBe(SIZES.partHeight - 1)
  })

  it('carries what was left over into the next one', () => {
    const first = woundBy(hangMany(), pixels(12), 0)
    const next = woundBy(hangMany(), pixels(12), first.left)
    expect(first.by).toBe(0)
    expect(next.by).toBe(1)
    expect(next.left).toBe(4)
  })

  it('is one part for a flick of a hand, not one for every event it sends', () => {
    // A trackpad gives a few pixels at a time, dozens of times a flick.
    let carried = 0
    let wound = 0
    for (let at = 0; at < 8; at++) {
      const wheel = woundBy(hangMany(), pixels(3), carried)
      carried = wheel.left
      wound += wheel.by
    }
    expect(wound).toBe(1)
  })

  it('reads a wheel said in lines as one part the line', () => {
    expect(woundBy(hangMany(), { delta: 2, mode: 1 }, 0).by).toBe(2)
  })

  it('reads a wheel said in windows as the whole window', () => {
    const held = hangMany()
    expect(woundBy(held, { delta: 1, mode: 2 }, 0).by).toBe(held.shown)
  })

  it('winds back the way it came, and carries the leftover the same way', () => {
    const back = woundBy(hangMany(), pixels(-SIZES.partHeight - 6), 0)
    expect(back.by).toBe(-1)
    expect(back.left).toBe(-6)
  })
})

describe('however many parts stand at once', () => {
  /** A node with depth under it for as many parts as the ceiling allows. */
  const under = (ceiling: number) =>
    hangParts(NODE, parts(ceiling + 4), { ...SIZES, maxParts: ceiling }, DEPS)!

  it('every one of them is up by the time it is all the way open', () => {
    // A lead that outran the opening left the last of them at nothing at all,
    // on a ground drawn deep enough to hold them.
    for (const ceiling of [1, 2, 6, 9, 12, 20]) {
      const settled = under(ceiling)
      const drawn = getOpenParts(settled, 1)!.parts
      expect(drawn).toHaveLength(settled.shown)
      for (const part of drawn) expect(part.opacity).toBe(1)
    }
  })

  it('the first of them is still ahead of the last partway through', () => {
    for (const ceiling of [2, 6, 12, 20]) {
      const drawn = getOpenParts(under(ceiling), 0.5)!.parts
      expect(drawn[0]!.opacity).toBeGreaterThan(drawn.at(-1)!.opacity)
    }
  })

  it('one alone opens with the whole of the opening to itself', () => {
    expect(getOpenParts(under(1), 0.5)!.parts[0]!.opacity).toBe(easeOut(0.5))
  })
})
