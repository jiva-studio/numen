/**
 * The menu on a node: what it offers, and what a choice comes to.
 *
 * What it offers is every command over a note. A person who never opens the
 * palette reaches these here, so what the two lists hold is asked of both.
 */
import { describe, expect, it, vi } from 'vitest'

import { ITEMS, chose } from './menu'
import { commandsOf, overNote } from '../commanding'
import { WORDS as words } from '../words'

describe('what the menu offers', () => {
  it('offers every command over a note, in the order the commands are drawn', () => {
    expect(ITEMS.map((item) => item.id)).toStrictEqual(
      overNote(commandsOf(words)).map((one) => one.id),
    )
  })

  it('offers making a note, changing its title and removing it', () => {
    const offered = new Set(ITEMS.map((item) => item.id))

    expect([...offered].filter((id) => ['child', 'title', 'remove'].includes(id))).toHaveLength(3)
  })

  it('says what each of them does', () => {
    for (const item of ITEMS) expect(item.text).not.toBe('')
  })
})

describe('choosing one', () => {
  it('hands the command, the note and the name it is called by to the window', () => {
    for (const item of ITEMS) {
      const runs = vi.fn()
      chose(item.id, 'physics/Ontology.md', 'Ontology', { runs })

      expect(runs).toHaveBeenCalledWith(item.id, 'physics/Ontology.md', 'Ontology')
    }
  })

  it('does nothing for an identifier the menu does not offer', () => {
    const runs = vi.fn()

    chose('constructor', 'Ontology.md', 'Ontology', { runs })
    chose('', 'Ontology.md', 'Ontology', { runs })
    chose('destroy', 'Ontology.md', 'Ontology', { runs })

    expect(runs).not.toHaveBeenCalled()
  })
})
