/**
 * The decisions a palette makes, as plain values.
 *
 * Two of them are the reason this file exists. Where the keyboard lands is
 * counted over a list that is drawn in bands, so a number has to mean the same
 * thing to the arrow keys and to the drawing. And a band arrives while a person
 * is reading, so what the keyboard is on has to survive the list changing under
 * it.
 */
import { describe, expect, it } from 'vitest'
import {
  actionAt,
  choosable,
  flatten,
  keptAt,
  ordered,
  partsOf,
  placePalette,
  stepTo,
  type PaletteItem,
  type PaletteSection,
} from './model'

const OPEN = [{ id: 'open', text: 'Open' }]
const BOTH = [
  { id: 'travel', text: 'Show in plex' },
  { id: 'open', text: 'Open the note' },
]

const item = (id: string, more: Partial<PaletteItem> = {}): PaletteItem => ({
  id,
  title: id,
  actions: OPEN,
  ...more,
})

const band = (id: string, items: PaletteItem[], more: Partial<PaletteSection> = {}): PaletteSection => ({
  id,
  title: id,
  items,
  ...more,
})

/** Two bands, four items, and one of them not to be landed on. */
const SECTIONS: PaletteSection[] = [
  band('names', [item('one'), item('two')]),
  band('text', [item('three', { disabled: true }), item('four')]),
]

describe('one list drawn in bands', () => {
  it('numbers the items across the bands in the order they are drawn', () => {
    expect(flatten(SECTIONS).map((place) => place.item.id)).toEqual([
      'one',
      'two',
      'three',
      'four',
    ])
  })

  it('gives the drawing the same numbers the keyboard counts in', () => {
    const drawn = placePalette(SECTIONS).flatMap((placed) => placed.items)
    const walked = flatten(SECTIONS)

    expect(drawn.map((placed) => placed.at)).toEqual(walked.map((_, at) => at))
    expect(drawn.map((placed) => placed.item.id)).toEqual(walked.map((p) => p.item.id))
  })

  it('splits every line it draws, so the view works nothing out', () => {
    const placed = placePalette([
      band('names', [
        item('one', { title: 'Entropy', at: [{ from: 0, to: 3 }], detail: 'in: Ent' }),
      ]),
    ])

    expect(placed[0]?.items[0]?.name).toEqual([
      { text: 'Ent', hit: true },
      { text: 'ropy', hit: false },
    ])
    expect(placed[0]?.items[0]?.detail).toEqual([{ text: 'in: Ent', hit: false }])
  })

  it('draws no second line for an item that has none', () => {
    expect(placePalette([band('names', [item('one')])])[0]?.items[0]?.detail).toEqual([])
  })
})

describe('what may be landed on', () => {
  it('passes over an item that is turned off', () => {
    expect(choosable(item('one', { disabled: true }))).toBe(false)
  })

  it('passes over an item with nothing that can be done to it', () => {
    expect(choosable({ id: 'one', title: 'one' })).toBe(false)
    expect(choosable({ id: 'one', title: 'one', actions: [] })).toBe(false)
  })

  it('lands on an item that offers something', () => {
    expect(choosable(item('one'))).toBe(true)
  })
})

describe('walking the list', () => {
  const places = flatten(SECTIONS)

  it('crosses a band without stopping between them', () => {
    expect(stepTo(places, 1, 1)).toBe(3)
  })

  it('passes over what cannot be chosen on the way back too', () => {
    expect(stepTo(places, 3, -1)).toBe(1)
  })

  it('wraps at either end', () => {
    expect(stepTo(places, 3, 1)).toBe(0)
    expect(stepTo(places, 0, -1)).toBe(3)
  })

  it('is asked for the first by counting from nowhere', () => {
    expect(stepTo(places, -1, 1)).toBe(0)
  })

  it('is asked for the last by counting back from the first', () => {
    expect(stepTo(places, 0, -1)).toBe(3)
  })

  it('lands nowhere when nothing in the list can be chosen', () => {
    const off = flatten([band('names', [item('one', { disabled: true })])])
    expect(stepTo(off, -1, 1)).toBe(-1)
  })

  it('lands nowhere in a list with nothing in it', () => {
    expect(stepTo([], -1, 1)).toBe(-1)
  })
})

describe('a band arriving under the keyboard', () => {
  it('keeps the item it was on, wherever the arriving answers put it', () => {
    const before = flatten([band('names', [item('one')])])
    const after = flatten([
      band('names', [item('nought'), item('one')]),
      band('text', [item('two')]),
    ])

    expect(keptAt(before, 'one')).toBe(0)
    expect(keptAt(after, 'one')).toBe(1)
  })

  it('keeps the keyboard where it stood when the item it was on is gone', () => {
    const places = flatten(SECTIONS)
    expect(keptAt(places, 'vanished', 3)).toBe(3)
  })

  it('hands it to the first item when it stood nowhere', () => {
    expect(keptAt(flatten(SECTIONS), 'vanished', -1)).toBe(0)
  })

  it('hands it on when the item it was on is still there and turned off', () => {
    expect(keptAt(flatten(SECTIONS), 'three')).toBe(0)
  })

  it('takes the keyboard nowhere when nothing is left to land on', () => {
    expect(keptAt([], 'one')).toBe(-1)
  })
})

describe('what a key reaches', () => {
  it('reaches the first action, and the second with Shift', () => {
    const both = item('one', { actions: BOTH })
    expect(actionAt(both, false)).toBe('travel')
    expect(actionAt(both, true)).toBe('open')
  })

  it('reaches nothing with Shift when only one action is offered', () => {
    expect(actionAt(item('one'), true)).toBe('')
  })

  it('reaches nothing on an item that cannot be chosen', () => {
    expect(actionAt(item('one', { disabled: true, actions: BOTH }), false)).toBe('')
    expect(actionAt(undefined, false)).toBe('')
  })
})

describe('marking why an item is here', () => {
  it('draws a line nothing stands in as one run', () => {
    expect(partsOf('Entropy')).toEqual([{ text: 'Entropy', hit: false }])
  })

  it('draws nothing at all for a line with nothing on it', () => {
    expect(partsOf('', [{ from: 0, to: 3 }])).toEqual([])
  })

  it('marks a run at the start, in the middle and at the end', () => {
    expect(partsOf('abcdef', [{ from: 0, to: 2 }])).toEqual([
      { text: 'ab', hit: true },
      { text: 'cdef', hit: false },
    ])
    expect(partsOf('abcdef', [{ from: 2, to: 4 }])).toEqual([
      { text: 'ab', hit: false },
      { text: 'cd', hit: true },
      { text: 'ef', hit: false },
    ])
    expect(partsOf('abcdef', [{ from: 4, to: 6 }])).toEqual([
      { text: 'abcd', hit: false },
      { text: 'ef', hit: true },
    ])
  })

  it('marks the whole line as one run when the whole line is why', () => {
    expect(partsOf('abc', [{ from: 0, to: 3 }])).toEqual([{ text: 'abc', hit: true }])
  })

  it('draws two runs that do not touch as two', () => {
    expect(partsOf('abcdef', [
      { from: 0, to: 1 },
      { from: 4, to: 5 },
    ])).toEqual([
      { text: 'a', hit: true },
      { text: 'bcd', hit: false },
      { text: 'e', hit: true },
      { text: 'f', hit: false },
    ])
  })

  it('reads the runs in whatever order they arrive', () => {
    expect(partsOf('abcdef', [
      { from: 4, to: 5 },
      { from: 0, to: 1 },
    ])).toEqual([
      { text: 'a', hit: true },
      { text: 'bcd', hit: false },
      { text: 'e', hit: true },
      { text: 'f', hit: false },
    ])
  })

  it('folds runs that overlap or touch into one', () => {
    expect(partsOf('abcdef', [
      { from: 0, to: 3 },
      { from: 2, to: 4 },
    ])).toEqual([
      { text: 'abcd', hit: true },
      { text: 'ef', hit: false },
    ])
    expect(partsOf('abcdef', [
      { from: 0, to: 2 },
      { from: 2, to: 4 },
    ])).toEqual([
      { text: 'abcd', hit: true },
      { text: 'ef', hit: false },
    ])
  })

  it('reads a run given back to front', () => {
    expect(partsOf('abcdef', [{ from: 4, to: 2 }])).toEqual([
      { text: 'ab', hit: false },
      { text: 'cd', hit: true },
      { text: 'ef', hit: false },
    ])
  })

  it('brings a run that runs off either end back inside the line', () => {
    expect(partsOf('abc', [{ from: -5, to: 99 }])).toEqual([{ text: 'abc', hit: true }])
  })

  it('drops a run with nothing in it', () => {
    expect(partsOf('abc', [{ from: 1, to: 1 }])).toEqual([{ text: 'abc', hit: false }])
  })

  it('never ends a run on half a character', () => {
    // The emoji is two code units, and the run names one of them.
    const text = 'a👋b'
    expect(partsOf(text, [{ from: 1, to: 2 }])).toEqual([
      { text: 'a', hit: false },
      { text: '👋', hit: true },
      { text: 'b', hit: false },
    ])
    expect(partsOf(text, [{ from: 2, to: 3 }])).toEqual([
      { text: 'a', hit: false },
      { text: '👋', hit: true },
      { text: 'b', hit: false },
    ])
  })
})

describe('which band stands where', () => {
  const holding = band('names', [item('one')])
  const empty = band('text', [])
  const alsoEmpty = band('meaning', [])

  it('keeps the bands holding something in the order they were offered', () => {
    const also = band('text', [item('two')])
    expect(ordered([holding, also]).map((one) => one.id)).toEqual(['names', 'text'])
    expect(ordered([also, holding]).map((one) => one.id)).toEqual(['text', 'names'])
  })

  it('sends a band holding nothing to the foot', () => {
    expect(ordered([empty, holding]).map((one) => one.id)).toEqual(['names', 'text'])
  })

  it('keeps the bands holding nothing in the order they were offered', () => {
    expect(ordered([empty, holding, alsoEmpty]).map((one) => one.id)).toEqual([
      'names',
      'text',
      'meaning',
    ])
  })

  it('leaves what the keyboard counts exactly where it was', () => {
    const sections = [empty, holding, alsoEmpty, band('more', [item('two'), item('three')])]
    expect(flatten(ordered(sections)).map((place) => place.item.id)).toEqual(
      flatten(sections).map((place) => place.item.id),
    )
  })
})
