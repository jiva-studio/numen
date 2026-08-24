/**
 * The menu on a node: what it offers, and what a choice comes to.
 */
import { describe, expect, it, vi } from 'vitest'

import { ITEMS, chose } from './menu'

const choices = () => ({
  open: vi.fn(),
  child: vi.fn(),
  ask: vi.fn(),
  copy: vi.fn(),
})

describe('what the menu offers', () => {
  it('offers four things, in the order they are drawn', () => {
    expect(ITEMS.map((item) => item.id)).toStrictEqual(['open', 'child', 'ask', 'copy'])
  })

  it('says what each of them does', () => {
    for (const item of ITEMS) expect(item.text).not.toBe('')
  })
})

describe('choosing one', () => {
  it('hands the note it was asked for on to the item chosen, and to no other', () => {
    for (const item of ITEMS) {
      const on = choices()
      chose(item.id, 'physics/Ontology.md', on)

      const called = Object.entries(on).filter(([, what]) => what.mock.calls.length > 0)
      expect(called.map(([id]) => id)).toStrictEqual([item.id])
      expect(called[0]?.[1]).toHaveBeenCalledWith('physics/Ontology.md')
    }
  })

  it('does nothing for an identifier the menu does not offer', () => {
    const on = choices()
    chose('constructor', 'Ontology.md', on)
    chose('', 'Ontology.md', on)

    for (const what of Object.values(on)) expect(what).not.toHaveBeenCalled()
  })
})
