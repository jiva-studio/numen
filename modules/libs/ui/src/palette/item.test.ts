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
  commandKeyChord,
  flatten,
  keptAt,
  keptOn,
  keyed,
  keyChord,
  opensActions,
  ordered,
  overlayIcon,
  partsOf,
  placeActions,
  placePalette,
  stepIn,
  stepTo,
  type PaletteItem,
  type PaletteBand,
} from './item'
import { MANY } from './fixtures/actions'

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

const band = (id: string, items: PaletteItem[], more: Partial<PaletteBand> = {}): PaletteBand => ({
  id,
  title: id,
  items,
  ...more,
})

/** Two bands, four items, and one of them not to be landed on. */
const SECTIONS: PaletteBand[] = [
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

  it('hands the keyboard to the first item when the one it was on is gone', () => {
    expect(keptAt(flatten(SECTIONS), 'vanished')).toBe(0)
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

  it('hands out a key each, in the order the actions are offered', () => {
    expect(keyed(item('one', { actions: BOTH }))).toEqual([
      { action: BOTH[0], key: { icons: ['return'], letter: '' } },
      { action: BOTH[1], key: { icons: ['shift', 'return'], letter: '' } },
    ])
  })

  it('says nothing of the actions past the keys', () => {
    expect(keyed(item('one', { actions: MANY })).map((one) => one.action.id)).toEqual([
      'travel',
      'open',
    ])
  })

  it('says nothing at all of an item that cannot be chosen', () => {
    expect(keyed(item('one', { disabled: true, actions: MANY }))).toEqual([])
    expect(keyed(undefined)).toEqual([])
  })
})

describe('the keystroke that opens the action panel', () => {
  const chord = (
    more: Partial<{ key: string; ctrlKey: boolean; metaKey: boolean; shiftKey: boolean }>,
  ) => ({
    key: 'k',
    ctrlKey: false,
    metaKey: false,
    shiftKey: false,
    ...more,
  })

  it('is K held with either of the two keys that hold a chord', () => {
    expect(opensActions(chord({ ctrlKey: true }))).toBe(true)
    expect(opensActions(chord({ metaKey: true }))).toBe(true)
  })

  it('is the same key in either case', () => {
    expect(opensActions(chord({ key: 'K', ctrlKey: true }))).toBe(true)
  })

  it('is not the letter on its own, and not another letter', () => {
    expect(opensActions(chord({}))).toBe(false)
    expect(opensActions(chord({ key: 'j', ctrlKey: true }))).toBe(false)
  })

  it('is not the same chord with Shift held, which belongs to whoever takes it', () => {
    expect(opensActions(chord({ ctrlKey: true, shiftKey: true }))).toBe(false)
    expect(opensActions(chord({ metaKey: true, shiftKey: true }))).toBe(false)
  })

  it('is held with the key the keyboard in hand puts beside the space bar', () => {
    expect(commandKeyChord('MacIntel')).toEqual({ icons: ['command'], letter: 'K' })
    expect(commandKeyChord('Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)')).toEqual({
      icons: ['command'],
      letter: 'K',
    })
    expect(commandKeyChord('Linux x86_64')).toEqual({ icons: ['control'], letter: 'K' })
  })
})

describe('a keystroke of one letter and the key beside the space bar', () => {
  it('is held with the key the keyboard in hand puts there', () => {
    expect(overlayIcon('MacIntel')).toBe('command')
    expect(overlayIcon('Linux x86_64')).toBe('control')
    expect(keyChord('g', 'MacIntel')).toEqual({ icons: ['command'], letter: 'G' })
    expect(keyChord('g', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('is drawn in capitals, whichever case it was named in', () => {
    expect(keyChord('G', 'Linux x86_64')).toEqual({ icons: ['control'], letter: 'G' })
  })

  it('carries Shift between the two where the chord holds it', () => {
    expect(keyChord('p', 'MacIntel', true)).toEqual({
      icons: ['command', 'shift'],
      letter: 'P',
    })
    expect(keyChord('p', 'Linux x86_64', true)).toEqual({
      icons: ['control', 'shift'],
      letter: 'P',
    })
  })
})

describe('walking a list where every row can be landed on', () => {
  it('wraps at either end', () => {
    expect(stepIn(3, 2, 1)).toBe(0)
    expect(stepIn(3, 0, -1)).toBe(2)
  })

  it('is asked for the first by counting from nowhere', () => {
    expect(stepIn(3, -1, 1)).toBe(0)
  })

  it('lands nowhere in a list with nothing in it', () => {
    expect(stepIn(0, -1, 1)).toBe(-1)
  })
})

describe('the actions the panel draws', () => {
  it('draws every action when nothing has been typed', () => {
    expect(placeActions(MANY).map((one) => one.action.id)).toEqual([
      'travel',
      'open',
      'beside',
      'rename',
      'remove',
    ])
  })

  it('numbers them as they are drawn, so one number says which action', () => {
    expect(placeActions(MANY, 'open').map((one) => one.at)).toEqual([0, 1])
  })

  it('keeps only the actions the words are in, wherever in the name they stand', () => {
    expect(placeActions(MANY, 'note').map((one) => one.action.id)).toEqual(['open'])
    expect(placeActions(MANY, 'OPEN').map((one) => one.action.id)).toEqual(['open', 'beside'])
    expect(placeActions(MANY, '  plex  ').map((one) => one.action.id)).toEqual(['travel'])
  })

  it('keeps nothing when no name holds the words', () => {
    expect(placeActions(MANY, 'nowhere')).toEqual([])
  })

  it('marks the run the words stand in, and marks nothing where nothing was typed', () => {
    expect(placeActions(MANY, 'name')[0]?.name).toEqual([
      { text: 'Re', hit: false },
      { text: 'name', hit: true },
    ])
    expect(placeActions(MANY)[3]?.name).toEqual([{ text: 'Rename', hit: false }])
  })

  it('carries the key of an action a key reaches, and nothing for the rest', () => {
    expect(placeActions(MANY).map((one) => one.key)).toEqual([
      { icons: ['return'], letter: '' },
      { icons: ['shift', 'return'], letter: '' },
      null,
      null,
      null,
    ])
  })

  it('leaves an action its key wherever the words put it', () => {
    expect(placeActions(MANY, 'open')).toMatchObject([
      { action: { id: 'open' }, at: 0, key: { icons: ['shift', 'return'] } },
      { action: { id: 'beside' }, at: 1, key: null },
    ])
  })

  it('draws nothing for an item offering nothing', () => {
    expect(placeActions()).toEqual([])
    expect(placeActions([], 'open')).toEqual([])
  })
})

describe('a list of actions changing under the panel', () => {
  it('keeps the action it was on, wherever the words put it', () => {
    expect(keptOn(placeActions(MANY, 'open'), 'beside')).toBe(1)
    expect(keptOn(placeActions(MANY), 'beside')).toBe(2)
  })

  it('keeps it across a list offered again in arrays of its own', () => {
    const fresh = MANY.map((one) => ({ ...one }))

    expect(keptOn(placeActions(fresh), 'rename')).toBe(3)
  })

  it('hands it to the first action when the one it was on is gone', () => {
    expect(keptOn(placeActions(MANY, 'plex'), 'rename')).toBe(0)
  })

  it('takes it nowhere in a list holding none', () => {
    expect(keptOn([], 'rename')).toBe(-1)
    expect(keptOn(placeActions(MANY, 'nowhere'), 'rename')).toBe(-1)
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
  const empty = band('text', [], { silence: 'nothing to search with' })
  const alsoEmpty = band('meaning', [], { working: true })

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

  it('draws a band that answered with nothing nowhere', () => {
    const answered = band('text', [])
    expect(ordered([holding, answered]).map((one) => one.id)).toEqual(['names'])
    expect(ordered([answered]).map((one) => one.id)).toEqual([])
  })

  it('draws a band holding nothing while it is still working', () => {
    expect(ordered([band('meaning', [], { working: true })]).map((one) => one.id)).toEqual([
      'meaning',
    ])
  })

  it('draws a band that could not be asked, with what it has to say', () => {
    const notAsked = band('meaning', [], { silence: 'the vault could not answer' })
    expect(ordered([notAsked]).map((one) => one.id)).toEqual(['meaning'])
  })

  it('leaves what the keyboard counts exactly where it was', () => {
    const bands = [empty, holding, alsoEmpty, band('more', [item('two'), item('three')])]
    expect(flatten(ordered(bands)).map((place) => place.item.id)).toEqual(
      flatten(bands).map((place) => place.item.id),
    )
  })
})
