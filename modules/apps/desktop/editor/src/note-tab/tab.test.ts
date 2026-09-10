/**
 * What a tab does with what it is told, asked without a browser.
 *
 * Every one of these has a failure a person sees as their own work going
 * wrong: a caret that jumps mid-sentence, a buffer emptied by a rename, an
 * answer from before the last keystroke put on the screen over it.
 */
import { describe, expect, it } from 'vitest'
import { dirty, markOf, opening, stateOf, tabAfter, waiting, type Effect, type Tab } from './tab'

/** A tab that has read its note and shows what it read. */
const tab = (over: Partial<Tab> = {}): Tab => ({
  path: 'Note.md',
  written: 'one',
  filePath: 'a1',
  shown: 'one',
  pendingWrite: null,
  hasPendingWrite: false,
  isDeleted: false,
  isStale: false,
  since: null,
  reading: 1,
  refused: null,
  ...over,
})

const kinds = (effects: readonly Effect[]) => effects.map((effect) => effect.kind)

/** A tab whose write was told the file carries another fingerprint. */
const overtaken = (over: Partial<Tab> = {}): Tab =>
  tabAfter(tab({ shown: 'mine', pendingWrite: 'mine', since: 10, ...over }), {
    kind: 'written',
    answer: { kind: 'changed' },
  }).tab


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
      pendingWrite: 'typed',
      refused: 'tooLarge',
    })
    const readable = { ...everything, refused: null }

    expect(stateOf(everything)).toBe('stuck')
    expect(stateOf(readable)).toBe('loading')
    expect(stateOf({ ...readable, written: 'one', isStale: true })).toBe('stale')
    expect(stateOf({ ...readable, written: 'one' })).toBe('saving')
    expect(stateOf({ ...readable, written: 'one', pendingWrite: null })).toBe('unsaved')
    expect(stateOf(tab())).toBe('clean')
  })
})

describe('a read answers with a body', () => {
  it('fills a loading tab and leaves it clean', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'one', at: 'a1' },
    })

    expect(next.tab.written).toBe('one')
    expect(next.tab.filePath).toBe('a1')
    expect(next.tab.shown).toBe('one')
    expect(stateOf(next.tab)).toBe('clean')
    expect(next.effects).toEqual([{ kind: 'replace', body: 'one' }])
  })

  it('replaces the document when the body differs from what is shown', () => {
    const next = tabAfter(tab(), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'two', at: 'a2' },
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
      answer: { kind: 'body', body: 'one', at: 'a1' },
    })

    expect(next.effects).toEqual([])
    expect(next.tab).toEqual(tab())
  })

  it('is discarded while something is unsaved', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })
    const next = tabAfter(unsaved, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })

    expect(next.tab).toEqual(unsaved)
    expect(next.effects).toEqual([])
  })

  it('is discarded when it is older than the newest read issued', () => {
    const reading = tab({ reading: 3 })
    const next = tabAfter(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'stale', at: 'a2' },
    })

    expect(next.tab).toEqual(reading)
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
      { kind: 'write', path: 'Note.md', body: 'again', seen: null },
    ])
  })

  it('leaves the buffer alone out of a re-read, and says the note is not there', () => {
    // The note was removed or renamed under an open tab. What the person is
    // reading stays on the screen, and the tab stops writing it anywhere.
    const reading = tab({ reading: 2 })
    const next = tabAfter(reading, { kind: 'read', generation: 2, answer: { kind: 'missing' } })

    expect(next.tab.shown).toBe(reading.shown)
    expect(stateOf(next.tab)).toBe('gone')
    expect(next.effects).toEqual([])
  })

  it('leaves a tab standing on a live file alone where the read was overtaken', () => {
    // The note moved, the tab followed it and asked again. The read left behind
    // answers about the name the tab has left, and the file it is on is there.
    const reading = tab({ reading: 3 })
    const next = tabAfter(reading, { kind: 'read', generation: 2, answer: { kind: 'missing' } })

    expect(next).toEqual({ tab: reading, effects: [] })
    expect(stateOf(next.tab)).toBe('clean')
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

  it('leaves a loading tab alone where the read was overtaken', () => {
    const reading = tab({ written: null, shown: '', reading: 2 })
    const next = tabAfter(reading, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'refused', refusal: 'notText' },
    })

    expect(next).toEqual({ tab: reading, effects: [] })
    expect(stateOf(next.tab)).toBe('loading')
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
    const next = tabAfter(tab({ shown: 'one!', pendingWrite: 'one', since: 1000 }), {
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
    expect(next.tab.pendingWrite).toBe('two')
    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', path: 'a1' } }])
  })

  it('owes a write while one is in the air', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'fired',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
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
    const next = tabAfter(tab({ shown: 'two', pendingWrite: 'two', since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok', at: 'a2' },
    })

    expect(next.tab.written).toBe('two')
    expect(next.tab.filePath).toBe('a2')
    expect(next.tab.since).toBeNull()
    expect(stateOf(next.tab)).toBe('clean')
    expect(next.effects).toEqual([])
  })

  it('leaves the tab unsaved when the person typed while it was in the air', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok', at: 'a2' },
    })

    expect(next.tab.written).toBe('two')
    expect(dirty(next.tab)).toBe(true)
    expect(stateOf(next.tab)).toBe('unsaved')
  })

  it('begins the write it owed', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', hasPendingWrite: true, since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok', at: 'a2' },
    })

    expect(stateOf(next.tab)).toBe('saving')
    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'three', seen: { prose: 'two', path: 'a2' } },
    ])
  })
})

describe('the fingerprint the file was read at', () => {
  it('is what the next write presents, taken from the write that answered', () => {
    // Two saves with no read between. The file is what this tab's own write left
    // behind, and the fingerprint that write answered with is what it carries.
    const first = tabAfter(tabAfter(tab(), { kind: 'typed', body: 'two', at: 0 }).tab, {
      kind: 'fired',
    })
    expect(first.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', path: 'a1' } }])

    const landed = tabAfter(first.tab, { kind: 'written', answer: { kind: 'ok', at: 'a2' } })
    const again = tabAfter(landed.tab, { kind: 'typed', body: 'three', at: 100 })
    const second = tabAfter(again.tab, { kind: 'fired' })

    expect(second.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'three', seen: { prose: 'two', path: 'a2' } }])
  })

  it('is nothing for a file that was not there, and that write compares nothing', () => {
    const made = tabAfter(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    })

    expect(made.tab.filePath).toBeNull()
  })
})

describe('a write answers changed', () => {
  it('overtakes the tab and leaves what was shown where it is', () => {
    const next = tabAfter(tab({ shown: 'mine', pendingWrite: 'mine', since: 10 }), {
      kind: 'written',
      answer: { kind: 'changed' },
    })

    expect(stateOf(next.tab)).toBe('stale')
    expect(next.tab.pendingWrite).toBeNull()
    expect(next.tab.shown).toBe('mine')
    expect(next.tab.written).toBe('one')
    expect(next.effects).toEqual([])
  })

  it('drops the write that was owed', () => {
    const next = tabAfter(
      tab({ shown: 'mine', pendingWrite: 'was', hasPendingWrite: true, since: 10 }),
      { kind: 'written', answer: { kind: 'changed' } },
    )

    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.effects).toEqual([])
  })

  it('arms nothing while the person types into an overtaken tab', () => {
    const next = tabAfter(overtaken(), { kind: 'typed', body: 'mine and more', at: 2000 })

    expect(next.tab.shown).toBe('mine and more')
    expect(stateOf(next.tab)).toBe('stale')
    expect(next.effects).toEqual([])
  })

  it('writes nothing when an interval armed before it fires', () => {
    const stopped = overtaken()
    const next = tabAfter(stopped, { kind: 'fired' })

    expect(next.tab).toEqual(stopped)
    expect(next.effects).toEqual([])
  })

  it('reads nothing when the vault changes under it', () => {
    const stopped = overtaken()

    expect(tabAfter(stopped, { kind: 'changed', paths: ['Note.md'], renamed: [] })).toEqual({
      tab: stopped,
      effects: [],
    })
    expect(tabAfter(stopped, { kind: 'changed', paths: [], renamed: [] })).toEqual({
      tab: stopped,
      effects: [],
    })
  })

  it('holds a tab that the person types back to what was written', () => {
    const stopped = overtaken()
    const back = tabAfter(stopped, { kind: 'typed', body: 'one', at: 2000 }).tab

    expect(dirty(back)).toBe(false)
    expect(stateOf(back)).toBe('stale')
    expect(tabAfter(back, { kind: 'keeping' }).effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'one', seen: null },
    ])
  })

  it('discards an answer to a write it is no longer waiting on', () => {
    const stopped = overtaken()
    const next = tabAfter(stopped, {
      kind: 'written',
      answer: { kind: 'refused', refusal: 'tooLarge' },
    })

    expect(next.tab).toEqual(stopped)
    expect(stateOf(next.tab)).toBe('stale')
  })
})

describe('the person keeps theirs', () => {
  it('writes what is shown, comparing nothing', () => {
    const next = tabAfter(overtaken(), { kind: 'keeping' })

    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'mine', seen: null }])
    expect(stateOf(next.tab)).toBe('saving')
    expect(next.tab.isStale).toBe(false)
  })

  it('takes what that write answers with, and the tab is clean', () => {
    const writing = tabAfter(overtaken(), { kind: 'keeping' }).tab
    const next = tabAfter(writing, { kind: 'written', answer: { kind: 'ok', at: 'a3' } })

    expect(next.tab.written).toBe('mine')
    expect(next.tab.filePath).toBe('a3')
    expect(stateOf(next.tab)).toBe('clean')
  })

  it('does nothing to a tab that was not overtaken', () => {
    const unsaved = tab({ shown: 'two', since: 10 })

    expect(tabAfter(unsaved, { kind: 'keeping' })).toEqual({ tab: unsaved, effects: [] })
  })
})

describe("the person takes the file's", () => {
  it('reads again, and the tab stays overtaken until that read lands', () => {
    const next = tabAfter(overtaken(), { kind: 'taking' })

    expect(next.tab.reading).toBe(2)
    expect(stateOf(next.tab)).toBe('stale')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('replaces the buffer with what the read answers, unsaved as the tab is', () => {
    const reading = tabAfter(overtaken(), { kind: 'taking' }).tab
    const next = tabAfter(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })

    expect(next.effects).toEqual([{ kind: 'replace', body: 'theirs' }])
    expect(next.tab.shown).toBe('theirs')
    expect(next.tab.written).toBe('theirs')
    expect(next.tab.filePath).toBe('a2')
    expect(stateOf(next.tab)).toBe('clean')
  })

  it('leaves the bound to start again with the next change', () => {
    const typing = tabAfter(overtaken(), { kind: 'typed', body: 'mine and more', at: 1000 }).tab
    const reading = tabAfter(typing, { kind: 'taking' }).tab
    const took = tabAfter(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    }).tab

    expect(took.since).toBeNull()
    expect(tabAfter(took, { kind: 'typed', body: 'theirs and mine', at: 9000 }).effects).toEqual([
      { kind: 'arm', after: waiting.quiet },
    ])
  })

  it('stays overtaken for a read that is out of date', () => {
    const reading = tabAfter(overtaken(), { kind: 'taking' }).tab
    const next = tabAfter(reading, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'stale', at: 'a0' },
    })

    expect(next.tab).toEqual(reading)
    expect(stateOf(next.tab)).toBe('stale')
  })

  it('says the note is not there where the file is gone, since there is nothing to take', () => {
    const reading = tabAfter(overtaken(), { kind: 'taking' }).tab
    const next = tabAfter(reading, { kind: 'read', generation: 2, answer: { kind: 'missing' } })

    expect(next.tab.shown).toBe(reading.shown)
    expect(stateOf(next.tab)).toBe('gone')
  })

  it('does nothing to a tab that was not overtaken', () => {
    const clean = tab()

    expect(tabAfter(clean, { kind: 'taking' })).toEqual({ tab: clean, effects: [] })
  })
})

describe('a save is asked for', () => {
  it('writes what is unsaved now', () => {
    const next = tabAfter(tab({ shown: 'two', since: 10 }), { kind: 'saving' })

    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', path: 'a1' } }])
    expect(stateOf(next.tab)).toBe('saving')
  })

  it('owes one while a write is in the air', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'saving',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
    expect(next.effects).toEqual([])
  })

  it('writes nothing when nothing changed', () => {
    const clean = tab()

    expect(tabAfter(clean, { kind: 'saving' })).toEqual({ tab: clean, effects: [] })
  })

  it('writes nothing for a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })

    expect(tabAfter(loading, { kind: 'saving' })).toEqual({ tab: loading, effects: [] })
  })

  it('writes nothing while the tab is stuck', () => {
    const stuck = tab({ shown: 'two', refused: 'unreadable', since: 10 })

    expect(tabAfter(stuck, { kind: 'saving' })).toEqual({ tab: stuck, effects: [] })
  })

  it('writes nothing for an overtaken tab, which the person answers', () => {
    const stopped = overtaken()

    expect(tabAfter(stopped, { kind: 'saving' })).toEqual({ tab: stopped, effects: [] })
  })
})

describe('a file about to be renamed or removed', () => {
  it('lets go of the interval it is waiting on', () => {
    const clean = tab()

    expect(tabAfter(clean, { kind: 'settling' })).toEqual({
      tab: clean,
      effects: [{ kind: 'disarm' }],
    })
  })

  it('writes what is unsaved to the path it still stands at', () => {
    const next = tabAfter(tab({ shown: 'two', since: 10 }), { kind: 'settling' })

    expect(next.effects).toEqual([
      { kind: 'disarm' },
      { kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', path: 'a1' } },
    ])
    expect(stateOf(next.tab)).toBe('saving')
  })

  it('owes a write for what is unsaved behind one already in the air', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'settling',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
    expect(kinds(next.effects)).toEqual(['disarm'])
  })

  it('owes nothing behind a write that carries everything typed', () => {
    const next = tabAfter(tab({ pendingWrite: 'one' }), { kind: 'settling' })

    expect(next.tab.hasPendingWrite).toBe(false)
    expect(kinds(next.effects)).toEqual(['disarm'])
  })

  it('writes nothing for an overtaken tab, whose question is the person to answer', () => {
    const stopped = overtaken()
    const next = tabAfter(stopped, { kind: 'settling' })

    expect(next.tab).toEqual(stopped)
    expect(kinds(next.effects)).toEqual(['disarm'])
  })

  it('writes nothing for a tab whose note is no longer there', () => {
    const missing = tabAfter(tab({ reading: 1 }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    }).tab
    const next = tabAfter(missing, { kind: 'settling' })

    expect(next.tab).toEqual(missing)
    expect(kinds(next.effects)).toEqual(['disarm'])
  })

  it('leaves the tab where it is, so the note is followed wherever it went', () => {
    const settled = tabAfter(tab(), { kind: 'settling' }).tab
    const next = tabAfter(settled, {
      kind: 'changed',
      paths: [],
      renamed: [{ from: 'Note.md', to: 'Moved.md' }],
    })

    expect(next.tab.path).toBe('Moved.md')
  })
})

describe('a write answers a refusal', () => {
  it('sticks the tab, drops what was owed, and leaves the buffer editable', () => {
    const next = tabAfter(tab({ shown: 'two', pendingWrite: 'two', hasPendingWrite: true, since: 10 }), {
      kind: 'written',
      answer: { kind: 'refused', refusal: 'tooLarge' },
    })

    expect(stateOf(next.tab)).toBe('stuck')
    expect(next.tab.hasPendingWrite).toBe(false)
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
      renamed: [],
    })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('reads again for a reload, which names nothing at all', () => {
    const next = tabAfter(tab({ reading: 1 }), { kind: 'changed', paths: [], renamed: [] })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('does nothing to a tab with something unsaved, reload or not', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })

    expect(tabAfter(unsaved, { kind: 'changed', paths: ['Note.md'], renamed: [] })).toEqual({
      tab: unsaved,
      effects: [],
    })
    expect(tabAfter(unsaved, { kind: 'changed', paths: [], renamed: [] })).toEqual({
      tab: unsaved,
      effects: [],
    })
  })

  it('does nothing while loading, saving or stuck', () => {
    const held: Tab[] = [
      tab({ written: null, shown: '' }),
      tab({ shown: 'two', pendingWrite: 'two', since: 10 }),
      tab({ refused: 'unreadable' }),
    ]

    for (const one of held) {
      expect(tabAfter(one, { kind: 'changed', paths: [], renamed: [] })).toEqual({ tab: one, effects: [] })
    }
  })

  it('does nothing when the change names other paths only', () => {
    const clean = tab()
    const next = tabAfter(clean, { kind: 'changed', paths: ['Somewhere/Else.md'], renamed: [] })

    expect(next.tab).toEqual(clean)
    expect(next.effects).toEqual([])
  })
})

describe('a close is asked for', () => {
  it('writes what is unsaved and holds the tab', () => {
    const next = tabAfter(tab({ shown: 'two', since: 10 }), { kind: 'closing' })

    expect(kinds(next.effects)).toEqual(['write', 'hold'])
    expect(next.effects[0]).toEqual({ kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', path: 'a1' } })
    expect(stateOf(next.tab)).toBe('saving')
  })

  it('holds a tab with a write in the air, and owes it what was typed since', () => {
    const next = tabAfter(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'closing',
    })

    expect(next.effects).toEqual([{ kind: 'hold' }])
    expect(next.tab.hasPendingWrite).toBe(true)
  })

  it('closes a tab with nothing unsaved', () => {
    const next = tabAfter(tab(), { kind: 'closing' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('closes a tab that has not read its note, which owes nothing', () => {
    const next = tabAfter(tab({ written: null, shown: '' }), { kind: 'closing' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('holds an overtaken tab, writing nothing and leaving the question standing', () => {
    const next = tabAfter(overtaken(), { kind: 'closing' })

    expect(kinds(next.effects)).toEqual(['hold'])
    expect(stateOf(next.tab)).toBe('stale')
    expect(next.tab.shown).toBe('mine')
  })

  it('says why a stuck tab cannot be written before it goes', () => {
    const next = tabAfter(tab({ shown: 'two', refused: 'unreadable', since: 10 }), {
      kind: 'closing',
    })

    expect(next.effects).toEqual([{ kind: 'say', refusal: 'unreadable' }, { kind: 'close' }])
  })
})

describe('a note that moved', () => {
  it('is followed to where it went', () => {
    const next = tabAfter(tab(), {
      kind: 'changed',
      paths: ['Note.md', 'Renamed.md'],
      renamed: [{ from: 'Note.md', to: 'Renamed.md' }],
    })
    expect(next.tab.path).toBe('Renamed.md')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Renamed.md', generation: 2 }])
  })

  it('is followed even where the tab has something unwritten, so its save lands where the note is', () => {
    const next = tabAfter(tab({ shown: 'mine', since: 10 }), {
      kind: 'changed',
      paths: [],
      renamed: [{ from: 'Note.md', to: 'Renamed.md' }],
    })
    expect(next.tab.path).toBe('Renamed.md')
    // What the person typed is theirs, and a move is not a reason to read over it.
    expect(next.tab.shown).toBe('mine')
    expect(kinds(next.effects)).toEqual([])
  })

  it('leaves a tab that is not the one that moved where it is', () => {
    const next = tabAfter(tab(), {
      kind: 'changed',
      paths: ['Other.md'],
      renamed: [{ from: 'Other.md', to: 'Elsewhere.md' }],
    })
    expect(next.tab.path).toBe('Note.md')
    expect(kinds(next.effects)).toEqual([])
  })
})

describe('a note that is no longer there', () => {
  const vanished = (over: Partial<Tab> = {}) =>
    tabAfter(tab({ reading: 2, ...over }), {
      kind: 'read',
      generation: 2,
      answer: { kind: 'missing' },
    }).tab

  it('leaves what the person has on the screen', () => {
    expect(vanished({ shown: 'mine', since: 10 }).shown).toBe('mine')
  })

  it('is a state of its own, so the tab can say so', () => {
    expect(stateOf(vanished())).toBe('gone')
  })

  it('stops the unasked save, so nothing is written back without being asked', () => {
    const next = tabAfter(vanished({ shown: 'mine', since: 10 }), { kind: 'fired' })
    expect(kinds(next.effects)).toEqual([])
  })

  it('is not written by typing either', () => {
    const typed = tabAfter(vanished(), { kind: 'typed', body: 'more', at: 20 })
    expect(stateOf(typed.tab)).toBe('gone')
    expect(kinds(tabAfter(typed.tab, { kind: 'fired' }).effects)).toEqual([])
  })

  it('is made again at the name it had when the person says to keep it', () => {
    const next = tabAfter(vanished({ shown: 'mine', since: 10 }), { kind: 'keeping' })
    expect(next.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'mine', seen: null },
    ])
  })

  it('is itself again once the note comes back', () => {
    const back = tabAfter(vanished(), {
      kind: 'changed',
      paths: ['Note.md'],
      renamed: [],
    })
    const read = tabAfter(back.tab, {
      kind: 'read',
      generation: back.tab.reading,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })
    expect(stateOf(read.tab)).toBe('clean')
    expect(read.tab.shown).toBe('theirs')
  })
})

describe('the word a tab carries beside its title', () => {
  it('is nothing while the note is being read', () => {
    expect(markOf('loading')).toBeUndefined()
  })

  it('is nothing when the text on screen is the text of the file', () => {
    expect(markOf('clean')).toBeUndefined()
  })

  it('is unsaved while there is something to write', () => {
    expect(markOf('unsaved')).toBe('unsaved')
  })

  it('is unsaved while the write is on its way', () => {
    expect(markOf('saving')).toBe('unsaved')
  })

  it('is stale while the file has moved past what the tab read', () => {
    expect(markOf('stale')).toBe('stale')
  })

  it('is stuck when the note can be neither read nor written', () => {
    expect(markOf('stuck')).toBe('stuck')
  })

  it('is a different word for each of the three things a tab carries', () => {
    const words: (string | undefined)[] = [markOf('stuck'), markOf('stale'), markOf('unsaved')]
    expect(new Set(words).size).toBe(3)
  })
})
