/**
 * The decisions a palette makes, as plain values.
 *
 * Two of them are the reason this file exists. Where the keyboard lands is
 * counted over a list that is drawn in groups, so a number has to mean the same
 * thing to the arrow keys and to the drawing. And a group arrives while a person
 * is reading, so what the keyboard is on has to survive the list changing under
 * it.
 */
import { describe, expect, it } from 'vitest'
import { stepTo } from '@/shared/lib/roving'
import { findKeptAction, placeActions } from './actions'
import {
  canChoose,
  findKeptPlace,
  flatten,
  orderGroups,
  getSteppedIndex,
  canLandOn,
  type PaletteItem,
  type PaletteGroup,
} from './item'
import {
  getActionId,
  commandKeyChord,
  getShortcuts,
  isActionsChord,
  keyChord,
  overlayIcon,
} from './keys'
import { getParts } from './parts'
import { listId, optionId, placePalette } from './place'
import { MANY } from '../fixtures/actions'

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

const group = (
  id: string,
  items: PaletteItem[],
  more: Partial<PaletteGroup> = {},
): PaletteGroup => ({
  id,
  title: id,
  items,
  ...more,
})

/** Two groups, four items, and one of them not to be landed on. */
const SECTIONS: PaletteGroup[] = [
  group('names', [item('one'), item('two')]),
  group('text', [item('three', { isDisabled: true }), item('four')]),
]

describe('one list drawn in groups', () => {
  it('numbers the items across the groups in the order they are drawn', () => {
    expect(flatten(SECTIONS).map((place) => place.item.id)).toEqual(['one', 'two', 'three', 'four'])
  })

  it('gives the drawing the same numbers the keyboard counts in', () => {
    const paletteItems = placePalette(SECTIONS).flatMap((placed) => placed.items)
    const walked = flatten(SECTIONS)

    expect(paletteItems.map((placed) => placed.at)).toEqual(walked.map((_, at) => at))
    expect(paletteItems.map((placed) => placed.item.id)).toEqual(walked.map((p) => p.item.id))
  })

  it('splits every line it draws, so the view works nothing out', () => {
    const placedSections = placePalette([
      group('names', [
        item('one', { title: 'Entropy', at: [{ from: 0, to: 3 }], detail: 'in: Ent' }),
      ]),
    ])

    expect(placedSections[0]?.items[0]?.name).toEqual([
      { text: 'Ent', isHit: true },
      { text: 'ropy', isHit: false },
    ])
    expect(placedSections[0]?.items[0]?.detail).toEqual([{ text: 'in: Ent', isHit: false }])
  })

  it('draws no second line for an item that has none', () => {
    expect(placePalette([group('names', [item('one')])])[0]?.items[0]?.detail).toEqual([])
  })
})

describe('what may be landed on', () => {
  it('passes over an item that is turned off', () => {
    expect(canChoose(item('one', { isDisabled: true }))).toBe(false)
  })

  it('passes over an item with nothing that can be done to it', () => {
    expect(canChoose({ id: 'one', title: 'one' })).toBe(false)
    expect(canChoose({ id: 'one', title: 'one', actions: [] })).toBe(false)
  })

  it('lands on an item that offers something', () => {
    expect(canChoose(item('one'))).toBe(true)
  })
})

describe('walking the list', () => {
  const places = flatten(SECTIONS)

  it('crosses a group without stopping between them', () => {
    expect(stepTo(places, { from: 1, by: 1 }, canLandOn)).toBe(3)
  })

  it('passes over what cannot be chosen on the way back too', () => {
    expect(stepTo(places, { from: 3, by: -1 }, canLandOn)).toBe(1)
  })

  it('wraps at either end', () => {
    expect(stepTo(places, { from: 3, by: 1 }, canLandOn)).toBe(0)
    expect(stepTo(places, { from: 0, by: -1 }, canLandOn)).toBe(3)
  })

  it('is requested for the first by counting from nowhere', () => {
    expect(stepTo(places, { from: -1, by: 1 }, canLandOn)).toBe(0)
  })

  it('is requested for the last by counting back from the first', () => {
    expect(stepTo(places, { from: 0, by: -1 }, canLandOn)).toBe(3)
  })

  it('lands nowhere when nothing in the list can be chosen', () => {
    const off = flatten([group('names', [item('one', { isDisabled: true })])])
    expect(stepTo(off, { from: -1, by: 1 }, canLandOn)).toBe(-1)
  })

  it('lands nowhere in a list with nothing in it', () => {
    expect(stepTo([], { from: -1, by: 1 }, canLandOn)).toBe(-1)
  })
})

describe('a group arriving under the keyboard', () => {
  it('keeps the item it was on, wherever the arriving answers put it', () => {
    const before = flatten([group('names', [item('one')])])
    const after = flatten([
      group('names', [item('nought'), item('one')]),
      group('text', [item('two')]),
    ])

    expect(findKeptPlace(before, 'one')).toBe(0)
    expect(findKeptPlace(after, 'one')).toBe(1)
  })

  it('passes the keyboard to the first item when the one it was on is gone', () => {
    expect(findKeptPlace(flatten(SECTIONS), 'vanished')).toBe(0)
  })

  it('passes it on when the item it was on is still there and turned off', () => {
    expect(findKeptPlace(flatten(SECTIONS), 'three')).toBe(0)
  })

  it('takes the keyboard nowhere when nothing is left to land on', () => {
    expect(findKeptPlace([], 'one')).toBe(-1)
  })
})

describe('what a key reaches', () => {
  it('reaches the first action, and the second with Shift', () => {
    const both = item('one', { actions: BOTH })
    expect(getActionId(both, false)).toBe('travel')
    expect(getActionId(both, true)).toBe('open')
  })

  it('reaches nothing with Shift when only one action is offered', () => {
    expect(getActionId(item('one'), true)).toBe('')
  })

  it('reaches nothing on an item that cannot be chosen', () => {
    expect(getActionId(item('one', { isDisabled: true, actions: BOTH }), false)).toBe('')
    expect(getActionId(undefined, false)).toBe('')
  })

  it('passes out a key each, in the order the actions are offered', () => {
    expect(getShortcuts(item('one', { actions: BOTH }))).toEqual([
      { action: BOTH[0], key: { icons: ['return'], letter: '' } },
      { action: BOTH[1], key: { icons: ['shift', 'return'], letter: '' } },
    ])
  })

  it('says nothing of the actions past the keys', () => {
    expect(getShortcuts(item('one', { actions: MANY })).map((one) => one.action.id)).toEqual([
      'travel',
      'open',
    ])
  })

  it('says nothing at all of an item that cannot be chosen', () => {
    expect(getShortcuts(item('one', { isDisabled: true, actions: MANY }))).toEqual([])
    expect(getShortcuts(undefined)).toEqual([])
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

  it('is K retained with either of the two keys that hold a chord', () => {
    expect(isActionsChord(chord({ ctrlKey: true }))).toBe(true)
    expect(isActionsChord(chord({ metaKey: true }))).toBe(true)
  })

  it('is the same key in either case', () => {
    expect(isActionsChord(chord({ key: 'K', ctrlKey: true }))).toBe(true)
  })

  it('is not the letter on its own, and not another letter', () => {
    expect(isActionsChord(chord({}))).toBe(false)
    expect(isActionsChord(chord({ key: 'j', ctrlKey: true }))).toBe(false)
  })

  it('is not the same chord with Shift retained, which belongs to whoever takes it', () => {
    expect(isActionsChord(chord({ ctrlKey: true, shiftKey: true }))).toBe(false)
    expect(isActionsChord(chord({ metaKey: true, shiftKey: true }))).toBe(false)
  })

  it('is retained with the key the keyboard in hand puts alongside the space bar', () => {
    expect(commandKeyChord('MacIntel')).toEqual({ icons: ['command'], letter: 'K' })
    expect(commandKeyChord('Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)')).toEqual({
      icons: ['command'],
      letter: 'K',
    })
    expect(commandKeyChord('Linux x86_64')).toEqual({ icons: ['control'], letter: 'K' })
  })
})

describe('a keystroke of one letter and the key alongside the space bar', () => {
  it('is retained with the key the keyboard in hand puts there', () => {
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
    expect(getSteppedIndex(3, 2, 1)).toBe(0)
    expect(getSteppedIndex(3, 0, -1)).toBe(2)
  })

  it('is requested for the first by counting from nowhere', () => {
    expect(getSteppedIndex(3, -1, 1)).toBe(0)
  })

  it('lands nowhere in a list with nothing in it', () => {
    expect(getSteppedIndex(0, -1, 1)).toBe(-1)
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
      { text: 'Re', isHit: false },
      { text: 'name', isHit: true },
    ])
    expect(placeActions(MANY)[3]?.name).toEqual([{ text: 'Rename', isHit: false }])
  })

  it('marks the run when the words stand at the start of the name', () => {
    expect(placeActions(MANY, 'rena')[0]?.name).toEqual([
      { text: 'Rena', isHit: true },
      { text: 'me', isHit: false },
    ])
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
    expect(findKeptAction(placeActions(MANY, 'open'), 'beside')).toBe(1)
    expect(findKeptAction(placeActions(MANY), 'beside')).toBe(2)
  })

  it('keeps it across a list offered again in arrays of its own', () => {
    const cloned = MANY.map((one) => ({ ...one }))

    expect(findKeptAction(placeActions(cloned), 'rename')).toBe(3)
  })

  it('passes it to the first action when the one it was on is gone', () => {
    expect(findKeptAction(placeActions(MANY, 'plex'), 'rename')).toBe(0)
  })

  it('takes it nowhere in a list holding none', () => {
    expect(findKeptAction([], 'rename')).toBe(-1)
    expect(findKeptAction(placeActions(MANY, 'nowhere'), 'rename')).toBe(-1)
  })
})

describe('marking why an item is here', () => {
  it('draws a line nothing remains in as one run', () => {
    expect(getParts('Entropy')).toEqual([{ text: 'Entropy', isHit: false }])
  })

  it('draws nothing at all for a line with nothing on it', () => {
    expect(getParts('', [{ from: 0, to: 3 }])).toEqual([])
  })

  it('marks a run at the start, in the middle and at the end', () => {
    expect(getParts('abcdef', [{ from: 0, to: 2 }])).toEqual([
      { text: 'ab', isHit: true },
      { text: 'cdef', isHit: false },
    ])
    expect(getParts('abcdef', [{ from: 2, to: 4 }])).toEqual([
      { text: 'ab', isHit: false },
      { text: 'cd', isHit: true },
      { text: 'ef', isHit: false },
    ])
    expect(getParts('abcdef', [{ from: 4, to: 6 }])).toEqual([
      { text: 'abcd', isHit: false },
      { text: 'ef', isHit: true },
    ])
  })

  it('marks the whole line as one run when the whole line is why', () => {
    expect(getParts('abc', [{ from: 0, to: 3 }])).toEqual([{ text: 'abc', isHit: true }])
  })

  it('draws two runs that do not touch as two', () => {
    expect(
      getParts('abcdef', [
        { from: 0, to: 1 },
        { from: 4, to: 5 },
      ]),
    ).toEqual([
      { text: 'a', isHit: true },
      { text: 'bcd', isHit: false },
      { text: 'e', isHit: true },
      { text: 'f', isHit: false },
    ])
  })

  it('reads the runs in whatever order they arrive', () => {
    expect(
      getParts('abcdef', [
        { from: 4, to: 5 },
        { from: 0, to: 1 },
      ]),
    ).toEqual([
      { text: 'a', isHit: true },
      { text: 'bcd', isHit: false },
      { text: 'e', isHit: true },
      { text: 'f', isHit: false },
    ])
  })

  it('folds runs that overlap or touch into one', () => {
    expect(
      getParts('abcdef', [
        { from: 0, to: 3 },
        { from: 2, to: 4 },
      ]),
    ).toEqual([
      { text: 'abcd', isHit: true },
      { text: 'ef', isHit: false },
    ])
    expect(
      getParts('abcdef', [
        { from: 0, to: 2 },
        { from: 2, to: 4 },
      ]),
    ).toEqual([
      { text: 'abcd', isHit: true },
      { text: 'ef', isHit: false },
    ])
  })

  it('reads a run given back to front', () => {
    expect(getParts('abcdef', [{ from: 4, to: 2 }])).toEqual([
      { text: 'ab', isHit: false },
      { text: 'cd', isHit: true },
      { text: 'ef', isHit: false },
    ])
  })

  it('brings a run that runs off either end back inside the line', () => {
    expect(getParts('abc', [{ from: -5, to: 99 }])).toEqual([{ text: 'abc', isHit: true }])
  })

  it('drops a run with nothing in it', () => {
    expect(getParts('abc', [{ from: 1, to: 1 }])).toEqual([{ text: 'abc', isHit: false }])
  })

  it('never ends a run on half a character', () => {
    // The emoji is two code units, and the run names one of them.
    const text = 'a👋b'
    expect(getParts(text, [{ from: 1, to: 2 }])).toEqual([
      { text: 'a', isHit: false },
      { text: '👋', isHit: true },
      { text: 'b', isHit: false },
    ])
    expect(getParts(text, [{ from: 2, to: 3 }])).toEqual([
      { text: 'a', isHit: false },
      { text: '👋', isHit: true },
      { text: 'b', isHit: false },
    ])
  })
})

describe('which group remains where', () => {
  it('draws a group holding items once, whether or not it is working or has something to say', () => {
    const busy = group('names', [item('one')], { isWorking: true })
    const textGroup = group('text', [item('two')], { silence: 'nothing' })
    expect(orderGroups([busy, textGroup]).map((one) => one.id)).toEqual(['names', 'text'])
  })

  const itemGroup = group('names', [item('one')])
  const empty = group('text', [], { silence: 'nothing to search with' })
  const alsoEmpty = group('meaning', [], { isWorking: true })

  it('keeps the groups holding something in the order they were offered', () => {
    const also = group('text', [item('two')])
    expect(orderGroups([itemGroup, also]).map((one) => one.id)).toEqual(['names', 'text'])
    expect(orderGroups([also, itemGroup]).map((one) => one.id)).toEqual(['text', 'names'])
  })

  it('sends a group holding nothing to the foot', () => {
    expect(orderGroups([empty, itemGroup]).map((one) => one.id)).toEqual(['names', 'text'])
  })

  it('keeps the groups holding nothing in the order they were offered', () => {
    expect(orderGroups([empty, itemGroup, alsoEmpty]).map((one) => one.id)).toEqual([
      'names',
      'text',
      'meaning',
    ])
  })

  it('draws a group that answered with nothing nowhere', () => {
    const reply = group('text', [])
    expect(orderGroups([itemGroup, reply]).map((one) => one.id)).toEqual(['names'])
    expect(orderGroups([reply]).map((one) => one.id)).toEqual([])
  })

  it('draws a group holding nothing while it is still working', () => {
    expect(orderGroups([group('meaning', [], { isWorking: true })]).map((one) => one.id)).toEqual([
      'meaning',
    ])
  })

  it('draws a group that could not be requested, with what it has to say', () => {
    const notAsked = group('meaning', [], { silence: 'the vault could not answer' })
    expect(orderGroups([notAsked]).map((one) => one.id)).toEqual(['meaning'])
  })

  it('leaves what the keyboard counts exactly where it was', () => {
    const groups = [empty, itemGroup, alsoEmpty, group('more', [item('two'), item('three')])]
    expect(flatten(orderGroups(groups)).map((place) => place.item.id)).toEqual(
      flatten(groups).map((place) => place.item.id),
    )
  })
})

describe('where the half of a pair is moved off', () => {
  const HIGH_START = '\ud800'
  const HIGH_END = '\udbff'
  const LOW_START = '\udc00'
  const LOW_END = '\udfff'

  /** The run a span on the second unit of `first` + `second` picks out, in `x...y`. */
  const picked = (first: string, second: string) =>
    getParts(`x${first}${second}y`, [{ from: 2, to: 3 }])

  it('moves outwards from the middle of a pair at either end of the surrogate ranges', () => {
    for (const [first, second] of [
      [HIGH_START, LOW_START],
      [HIGH_END, LOW_END],
      [HIGH_START, LOW_END],
      [HIGH_END, LOW_START],
    ] as const) {
      expect(picked(first, second)).toEqual([
        { text: 'x', isHit: false },
        { text: first + second, isHit: true },
        { text: 'y', isHit: false },
      ])
    }
  })

  it('leaves a span alone when the unit before it does not open a pair', () => {
    for (const first of ['\ud7ff', LOW_START, 'a']) {
      expect(picked(first, LOW_START)).toEqual([
        { text: `x${first}`, isHit: false },
        { text: LOW_START, isHit: true },
        { text: 'y', isHit: false },
      ])
    }
  })

  it('leaves a span alone when the unit it starts on does not close a pair', () => {
    for (const second of [HIGH_END, '\ue000', 'b']) {
      expect(picked(HIGH_START, second)).toEqual([
        { text: `x${HIGH_START}`, isHit: false },
        { text: second, isHit: true },
        { text: 'y', isHit: false },
      ])
    }
  })
})

describe('what the list and its rows are addressed by', () => {
  it('names the list and each row after the id given', () => {
    expect(listId('p1')).toBe('p1-list')
    expect(optionId('p1', 3)).toBe('p1-option-3')
  })
})
