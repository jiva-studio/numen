/**
 * What a run of letters lands on. The clock is a number here, so the cases a
 * mounted menu cannot reach — a letter struck a second after the last one, a
 * word that runs off the end of the list — are plain arithmetic.
 */
import { describe, expect, it } from 'vitest'
import type { MenuItem } from './item'
import { isLetter, jumpTo, NOTHING_TYPED, type Typeahead } from './typeahead'

const ITEMS: MenuItem[] = [
  { id: 'open', text: 'Open' },
  { id: 'child', text: 'New child note' },
  { id: 'copy', text: 'Copy path' },
  { id: 'cut', text: 'Cut' },
]

/** A word typed one letter at a time, each letter a moment after the last. */
const types = (word: string): { typed: Typeahead; at: number | null } => {
  let typed = NOTHING_TYPED
  let at: number | null = null
  let now = 0
  for (const letter of word) {
    const jumped = jumpTo(ITEMS, typed, letter, at ?? -1, now)
    typed = jumped.typed
    at = jumped.at
    now += 10
  }
  return { typed, at }
}

describe('what a run of letters lands on', () => {
  it('lands on the first item the letter begins', () => {
    expect(types('c').at).toBe(2)
  })

  it('stands where it is while the word grows', () => {
    expect(types('co').at).toBe(2)
  })

  it('lands on the item the whole word begins rather than the first letter', () => {
    expect(types('cu').at).toBe(3)
  })

  it('walks the items one letter begins, and wraps', () => {
    const first = jumpTo(ITEMS, NOTHING_TYPED, 'c', -1, 0)
    const second = jumpTo(ITEMS, first.typed, 'c', first.at!, 10)
    const third = jumpTo(ITEMS, second.typed, 'c', second.at!, 20)
    expect([first.at, second.at, third.at]).toStrictEqual([2, 3, 2])
  })

  it('wraps to the top counting on from where the keyboard stands', () => {
    expect(jumpTo(ITEMS, NOTHING_TYPED, 'o', 2, 0).at).toBe(0)
  })

  it('lands on nothing where no item begins with the word', () => {
    expect(types('z').at).toBeNull()
  })

  it('passes over an item that cannot be chosen', () => {
    const items: MenuItem[] = [
      { id: 'copy', text: 'Copy path', disabled: true },
      { id: 'cut', text: 'Cut' },
    ]
    expect(jumpTo(items, NOTHING_TYPED, 'c', -1, 0).at).toBe(1)
  })

  it('lands on nothing where there is nothing to land on', () => {
    expect(jumpTo([], NOTHING_TYPED, 'c', -1, 0).at).toBeNull()
  })
})

/** A run of letters stays one word for a second. */
describe('how long a run of letters stays one word', () => {
  it('carries the word on while the letters keep coming', () => {
    const first = jumpTo(ITEMS, NOTHING_TYPED, 'c', -1, 1000)
    expect(jumpTo(ITEMS, first.typed, 'u', first.at!, 2000).typed.word).toBe('cu')
  })

  it('begins a word again once the run has gone quiet', () => {
    const first = jumpTo(ITEMS, NOTHING_TYPED, 'c', -1, 1000)
    const later = jumpTo(ITEMS, first.typed, 'o', first.at!, 2001)
    expect(later.typed.word).toBe('o')
    expect(later.at).toBe(0)
  })
})

describe('a key that stands for a letter', () => {
  const press = (over: Partial<KeyboardEvent> = {}) =>
    isLetter({ key: 'c', ctrlKey: false, metaKey: false, altKey: false, ...over })

  it('is a letter typed on its own', () => {
    expect(press()).toBe(true)
  })

  it('is not a chord', () => {
    expect(press({ ctrlKey: true })).toBe(false)
    expect(press({ metaKey: true })).toBe(false)
    expect(press({ altKey: true })).toBe(false)
  })

  it('is not the space bar, which belongs to the item the keyboard is on', () => {
    expect(press({ key: ' ' })).toBe(false)
  })

  it('is not a key named by a word', () => {
    expect(press({ key: 'ArrowDown' })).toBe(false)
  })
})
