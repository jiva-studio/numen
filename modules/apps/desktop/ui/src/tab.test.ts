/**
 * What a tab does with what it is told, asked without a browser.
 *
 * Every one of these has a failure a person sees as their own work going
 * wrong: a caret that jumps mid-sentence, a buffer emptied by a rename, an
 * answer from before the last keystroke put on the screen over it.
 */
import { describe, expect, it } from 'vitest'
import { dirty, opening, stateOf, tabAfter, waiting, type Effect, type Tab } from './tab'

/** A tab that has read its note and shows what it read. */
const tab = (over: Partial<Tab> = {}): Tab => ({
  path: 'Note.md',
  written: 'one',
  shown: 'one',
  flight: null,
  owed: false,
  since: null,
  reading: 1,
  refused: null,
  ...over,
})

const kinds = (effects: readonly Effect[]) => effects.map((effect) => effect.kind)

describe('a tab as it opens', () => {
  it('has nothing read yet and asks for the note', () => {
    const next = opening('Note.md')

    expect(stateOf(next.tab)).toBe('loading')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 1 }])
  })
})

describe('the states', () => {
  it('are read in order, so a tab that is all of them at once is the first', () => {
    const everything = tab({
      written: null,
      shown: 'typed',
      flight: { body: 'typed' },
      refused: 'tooLarge',
    })
    const readable = { ...everything, refused: null }

    expect(stateOf(everything)).toBe('stuck')
    expect(stateOf(readable)).toBe('loading')
    expect(stateOf({ ...readable, written: 'one' })).toBe('saving')
    expect(stateOf({ ...readable, written: 'one', flight: null })).toBe('unsaved')
    expect(stateOf(tab())).toBe('clean')
  })
})

describe('a read answers with a body', () => {
  it('fills a loading tab and leaves it clean', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'one' },
    })

    expect(next.tab.written).toBe('one')
    expect(next.tab.shown).toBe('one')
    expect(stateOf(next.tab)).toBe('clean')
    expect(next.effects).toEqual([{ kind: 'replace', body: 'one' }])
  })

  it('replaces the document when the body differs from what is shown', () => {
    const next = tabAfter(tab(), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'two' },
    })

    expect(next.effects).toEqual([{ kind: 'replace', body: 'two' }])
    expect(next.tab.written).toBe('two')
    expect(next.tab.shown).toBe('two')
  })

  it('replaces nothing when the body is what is already shown', () => {
    // The tab's own save coming back around. Replacing here collapses the caret
    // in the middle of a sentence.
    const next = tabAfter(tab(), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'one' },
    })

    expect(next.effects).toEqual([])
    expect(next.tab).toEqual(tab())
  })

  it('is discarded while something is unsaved', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })
    const next = tabAfter(unsaved, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'theirs' },
    })

    expect(next.tab).toEqual(unsaved)
    expect(next.effects).toEqual([])
  })

  it('is discarded when it is older than the newest read issued', () => {
    const asking = tab({ reading: 3 })
    const next = tabAfter(asking, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'stale' },
    })

    expect(next.tab).toEqual(asking)
    expect(next.effects).toEqual([])
  })
})

describe('a read answers missing', () => {
  it('empties a loading tab, and the next write creates the file', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    })

    expect(next.tab.written).toBe('')
    expect(next.tab.shown).toBe('')
    expect(stateOf(next.tab)).toBe('clean')

    const typed = tabAfter(next.tab, { kind: 'typed', body: 'again', at: 0 })
    expect(tabAfter(typed.tab, { kind: 'fired' }).effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'again' },
    ])
  })

  it('leaves the buffer alone out of a re-read', () => {
    // The note was renamed under an open tab. What the person is reading stays
    // on the screen at the name they opened.
    const reading = tab({ reading: 2 })
    const next = tabAfter(reading, { kind: 'read', generation: 2, answer: { kind: 'missing' } })

    expect(next.tab).toEqual(reading)
    expect(next.effects).toEqual([])
  })
})

describe('a read answers a refusal', () => {
  it('sticks a loading tab', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'refused', refusal: 'notText' },
    })

    expect(next.tab.refused).toBe('notText')
    expect(stateOf(next.tab)).toBe('stuck')
    expect(next.effects).toEqual([])
  })

  it('strands nothing: a tab that never read its note is not mended by typing', () => {
    // Clearing the refusal here would take the tab back to loading, where the
    // interval does nothing, and what the person typed would never be written.
    const stuck = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'refused', refusal: 'tooLarge' },
    }).tab

    const typed = tabAfter(stuck, { kind: 'typed', body: 'anything', at: 10 })

    expect(typed.tab.refused).toBe('tooLarge')
    expect(stateOf(typed.tab)).toBe('stuck')
    expect(tabAfter(typed.tab, { kind: 'fired' }).effects).toEqual([])
    expect(tabAfter(typed.tab, { kind: 'closing' }).effects).toEqual([
      { kind: 'say', refusal: 'tooLarge' },
      { kind: 'close' },
    ])
  })

  it('is discarded out of a re-read', () => {
    const reading = tab({ reading: 2 })
    const next = tabAfter(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'refused', refusal: 'tooLarge' },
    })

    expect(next.tab).toEqual(reading)
    expect(next.effects).toEqual([])
  })
})

describe('the person types', () => {
  it('changes what is shown, marks when it began, and arms the interval', () => {
    const next = tabAfter(tab(), { kind: 'typed', body: 'onemore', at: 1000 })

    expect(next.tab.shown).toBe('onemore')
    expect(next.tab.since).toBe(1000)
    expect(stateOf(next.tab)).toBe('unsaved')
    expect(next.effects).toEqual([{ kind: 'arm', after: waiting.quiet }])
  })

  it('keeps when the first unwritten change was made', () => {
    const next = tabAfter(tab({ shown: 'on', since: 1000 }), {
      kind: 'typed',
      body: 'one',
      at: 1200,
    })

    expect(next.tab.since).toBe(1000)
  })

  it('puts the write off while a write is in the air', () => {
    const next = tabAfter(tab({ shown: 'one!', flight: { body: 'one' }, since: 1000 }), {
      kind: 'typed',
      body: 'one!!',
      at: 1100,
    })

    expect(next.tab.shown).toBe('one!!')
    expect(stateOf(next.tab)).toBe('saving')
    expect(next.effects).toEqual([{ kind: 'arm', after: waiting.quiet }])
  })

  it('mends a refusal that is about the body', () => {
    const next = tabAfter(tab({ shown: 'huge', refused: 'tooLarge', since: 10 }), {
      kind: 'typed',
      body: 'small',
      at: 20,
    })

    expect(next.tab.refused).toBeNull()
    expect(stateOf(next.tab)).toBe('unsaved')
  })

  it('leaves a refusal that is a condition of the file', () => {
    const next = tabAfter(tab({ shown: 'edit', refused: 'unreadable', since: 10 }), {
      kind: 'typed',
      body: 'edited',
      at: 20,
    })

    expect(next.tab.refused).toBe('unreadable')
    expect(stateOf(next.tab)).toBe('stuck')
  })

  it('does nothing to a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })
    const next = tabAfter(loading, { kind: 'typed', body: 'early', at: 10 })

    expect(next.tab).toEqual(loading)
    expect(next.effects).toEqual([])
  })
})

describe('the interval fires', () => {
  it('begins a write of what is on screen when something is unsaved', () => {
    const next = tabAfter(tab({ shown: 'two', since: 10 }), { kind: 'fired' })

    expect(stateOf(next.tab)).toBe('saving')
    expect(next.tab.flight).toEqual({ body: 'two' })
    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'two' }])
  })

  it('owes a write while one is in the air', () => {
    const next = tabAfter(tab({ shown: 'three', flight: { body: 'two' }, since: 10 }), {
      kind: 'fired',
    })

    expect(next.tab.owed).toBe(true)
    expect(next.effects).toEqual([])
  })

  it('writes nothing when nothing changed', () => {
    const next = tabAfter(tab(), { kind: 'fired' })

    expect(next.tab).toEqual(tab())
    expect(next.effects).toEqual([])
  })

  it('writes nothing for a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })
    const next = tabAfter(loading, { kind: 'fired' })

    expect(next.tab).toEqual(loading)
    expect(next.effects).toEqual([])
  })

  it('writes nothing while the tab is stuck', () => {
    const stuck = tab({ shown: 'two', refused: 'unreadable', since: 10 })
    const next = tabAfter(stuck, { kind: 'fired' })

    expect(next.tab).toEqual(stuck)
    expect(next.effects).toEqual([])
  })

  it('is armed no later than the bound after the first unwritten change', () => {
    // Typing without a pause: every keystroke puts the write off, and the bound
    // is what makes one happen.
    let held = tab()
    let after = waiting.quiet
    for (let at = 0; at <= waiting.bound; at += 100) {
      const next = tabAfter(held, { kind: 'typed', body: `${at}`, at })
      held = next.tab
      after = next.effects[0]?.kind === 'arm' ? next.effects[0].after : after
      expect(at + after).toBeLessThanOrEqual(waiting.bound)
    }

    expect(after).toBe(0)
    expect(stateOf(tabAfter(held, { kind: 'fired' }).tab)).toBe('saving')
  })
})

describe('a write answers ok', () => {
  it('takes the body it carried as written, and forgets when the change was made', () => {
    const next = tabAfter(tab({ shown: 'two', flight: { body: 'two' }, since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok' },
    })

    expect(next.tab.written).toBe('two')
    expect(next.tab.since).toBeNull()
    expect(stateOf(next.tab)).toBe('clean')
    expect(next.effects).toEqual([])
  })

  it('leaves the tab unsaved when the person typed while it was in the air', () => {
    const next = tabAfter(tab({ shown: 'three', flight: { body: 'two' }, since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok' },
    })

    expect(next.tab.written).toBe('two')
    expect(dirty(next.tab)).toBe(true)
    expect(stateOf(next.tab)).toBe('unsaved')
  })

  it('begins the write it owed', () => {
    const next = tabAfter(tab({ shown: 'three', flight: { body: 'two' }, owed: true, since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok' },
    })

    expect(stateOf(next.tab)).toBe('saving')
    expect(next.tab.owed).toBe(false)
    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'three' }])
  })
})

describe('a write answers a refusal', () => {
  it('sticks the tab, drops what was owed, and leaves the buffer editable', () => {
    const next = tabAfter(tab({ shown: 'two', flight: { body: 'two' }, owed: true, since: 10 }), {
      kind: 'written',
      answer: { kind: 'refused', refusal: 'tooLarge' },
    })

    expect(stateOf(next.tab)).toBe('stuck')
    expect(next.tab.owed).toBe(false)
    expect(next.tab.shown).toBe('two')

    const typed = tabAfter(next.tab, { kind: 'typed', body: 'small', at: 20 })
    expect(typed.tab.shown).toBe('small')
  })
})

describe('the vault changes', () => {
  it('reads again when the change names this path and nothing is unsaved', () => {
    const next = tabAfter(tab({ reading: 1 }), {
      kind: 'changed',
      paths: ['Other.md', 'Note.md'],
    })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('reads again for a reload, which names nothing at all', () => {
    const next = tabAfter(tab({ reading: 1 }), { kind: 'changed', paths: [] })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('does nothing to a tab with something unsaved, reload or not', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })

    expect(tabAfter(unsaved, { kind: 'changed', paths: ['Note.md'] })).toEqual({
      tab: unsaved,
      effects: [],
    })
    expect(tabAfter(unsaved, { kind: 'changed', paths: [] })).toEqual({
      tab: unsaved,
      effects: [],
    })
  })

  it('does nothing while loading, saving or stuck', () => {
    const held: Tab[] = [
      tab({ written: null, shown: '' }),
      tab({ shown: 'two', flight: { body: 'two' }, since: 10 }),
      tab({ refused: 'unreadable' }),
    ]

    for (const one of held) {
      expect(tabAfter(one, { kind: 'changed', paths: [] })).toEqual({ tab: one, effects: [] })
    }
  })

  it('does nothing when the change names other paths only', () => {
    const clean = tab()
    const next = tabAfter(clean, { kind: 'changed', paths: ['Somewhere/Else.md'] })

    expect(next.tab).toEqual(clean)
    expect(next.effects).toEqual([])
  })
})

describe('a close is asked for', () => {
  it('writes what is unsaved and holds the tab', () => {
    const next = tabAfter(tab({ shown: 'two', since: 10 }), { kind: 'closing' })

    expect(kinds(next.effects)).toEqual(['write', 'hold'])
    expect(next.effects[0]).toEqual({ kind: 'write', path: 'Note.md', body: 'two' })
    expect(stateOf(next.tab)).toBe('saving')
  })

  it('holds a tab with a write in the air, and owes it what was typed since', () => {
    const next = tabAfter(tab({ shown: 'three', flight: { body: 'two' }, since: 10 }), {
      kind: 'closing',
    })

    expect(next.effects).toEqual([{ kind: 'hold' }])
    expect(next.tab.owed).toBe(true)
  })

  it('closes a tab with nothing unsaved', () => {
    const next = tabAfter(tab(), { kind: 'closing' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('closes a tab that has not read its note, which owes nothing', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), { kind: 'closing' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('says why a stuck tab cannot be written before it goes', () => {
    const next = tabAfter(tab({ shown: 'two', refused: 'unreadable', since: 10 }), {
      kind: 'closing',
    })

    expect(next.effects).toEqual([{ kind: 'say', refusal: 'unreadable' }, { kind: 'close' }])
  })
})
