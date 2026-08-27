/**
 * The parts of a node: which of them are drawn, how far each is set in, and
 * where each stands partway through the opening.
 */
import { describe, expect, it } from 'vitest'
import { MOST, REST, hangParts, openedTo, type PlexPart, type Room } from './inside'
import type { PlacedNode } from './model'

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

const SIZES = { partHeight: 20, partIndent: 10 }

/** A window with room to spare, and a measurer of ten pixels the letter. */
const ROOM: Room = {
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

const hung = (held: readonly PlexPart[], room = ROOM) => hangParts(NODE, held, SIZES, room)

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

describe('more parts than are drawn', () => {
  it('draws the ceiling and one more for the rest', () => {
    const held = hung(parts(MOST + 4))?.parts ?? []
    expect(held).toHaveLength(MOST + 1)
    expect(held.at(-1)?.text).toBe(REST)
  })

  it('leaves that one no identifier, so it is nowhere to go', () => {
    const held = hung(parts(MOST + 1))?.parts ?? []
    expect(held.at(-1)?.id).toBe('')
  })

  it('draws no such part where every one of them fits', () => {
    const held = hung(parts(MOST))?.parts ?? []
    expect(held).toHaveLength(MOST)
    expect(held.some((part) => part.id === '')).toBe(false)
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
  const opened = (open: number, count = 3) => openedTo(hung(parts(count))!, open)

  it('draws nothing while it is shut', () => {
    expect(opened(0)).toBeNull()
  })

  it('has its ground barely up before any of them has risen', () => {
    expect(opened(0.0001)?.opacity).toBeLessThan(0.01)
  })

  it('brings the ground up at the depth it keeps', () => {
    const settled = hung(parts(3))!
    for (const open of [0.2, 0.5, 1]) {
      expect(openedTo(settled, open)!.height).toBe(settled.height)
    }
    expect(openedTo(settled, 1)!.opacity).toBe(1)
  })

  it('has the first part further along than the last', () => {
    const drawn = opened(0.5)?.parts ?? []
    expect(drawn[0]!.opacity).toBeGreaterThan(drawn.at(-1)!.opacity)
  })

  it('rises onto its place from below, never from above it', () => {
    const settled = hung(parts(3))!
    openedTo(settled, 0.5)!.parts.forEach((part, at) => {
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
      for (const part of openedTo(settled, open)!.parts) {
        expect(part.y).toBeGreaterThanOrEqual(0)
        expect(part.y + settled.partHeight).toBeLessThanOrEqual(
          settled.height - 2 * settled.pad,
        )
      }
    }
  })

  it('rests every part where it hangs once it is all the way open', () => {
    const settled = hung(parts(3))!
    const drawn = openedTo(settled, 1)?.parts ?? []
    expect(drawn.map((part) => part.y)).toStrictEqual(settled.parts.map((part) => part.at))
    expect(drawn.every((part) => part.opacity === 1)).toBe(true)
  })

  it('brings the ground up before the parts have finished rising', () => {
    const settled = hung(parts(3))!
    const half = openedTo(settled, 0.5)!
    expect(half.opacity).toBeGreaterThan(half.parts.at(-1)!.opacity)
  })

  it('carries the words and the indent through untouched', () => {
    const part = opened(1)?.parts[1]
    expect(part?.text).toBe('Part 1')
    expect(part?.id).toBe('1')
    expect(part?.indent).toBe(0)
  })
})

describe('how wide the parts are drawn', () => {
  const wide = (held: readonly PlexPart[], room = ROOM) => hung(held, room)!

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
    const bare = { viewport: ROOM.viewport, margin: ROOM.margin }
    expect(wide(parts(3), bare).width).toBe(NODE.width)
  })

  it('is held inside the window, however long the words run', () => {
    const long = [{ id: '0', text: 'x'.repeat(400), level: 1 }]
    expect(wide(long).width).toBe(ROOM.viewport.width - 2 * ROOM.margin)
  })

  it('grows about the node’s own middle where the window has room', () => {
    expect(wide(parts(2)).offset).toBe(0)
  })

  it('slides back inside the window where the middle leaves no room', () => {
    const near = { ...NODE, x: 460 }
    const held = [{ id: '0', text: 'A heading of some length', level: 1 }]
    const settled = hangParts(near, held, SIZES, ROOM)!
    expect(near.x + settled.offset + settled.width / 2).toBeLessThanOrEqual(
      ROOM.viewport.width / 2 - ROOM.margin,
    )
  })
})
