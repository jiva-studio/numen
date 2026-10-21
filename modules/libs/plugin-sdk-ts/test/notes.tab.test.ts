/**
 * What a tab does with what it is told, asked without a browser.
 *
 * Every one of these has a failure a person sees as their own work going
 * wrong: a caret that jumps mid-sentence, a buffer emptied by a rename, an
 * answer from before the last keystroke put on the screen over it.
 */
import { describe, expect, it } from 'vitest'
import {
  isDirty,
  getMarkOf,
  openTab,
  getState,
  transitionTab,
  DEFAULT_WRITE_LIMITS,
  type Effect,
  type Tab,
} from '../src/notes/tab'

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
  error: null,
  ...over,
})

const kinds = (effects: readonly Effect[]) => effects.map((effect) => effect.kind)

/** A tab whose write was told the file carries another fingerprint. */
const createStaleTab = (over: Partial<Tab> = {}): Tab =>
  transitionTab(tab({ shown: 'mine', pendingWrite: 'mine', since: 10, ...over }), {
    kind: 'written',
    answer: { kind: 'changed' },
  }).tab

describe('a tab as it opens', () => {
  it('has nothing read yet and asks for the note', () => {
    const next = openTab('Note.md')

    expect(getState(next.tab)).toBe('loading')
    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 1 }])
  })
})

describe('the states', () => {
  it('are read in order, so a tab that is all of them at once is the first', () => {
    const everything = tab({
      written: null,
      shown: 'typed',
      pendingWrite: 'typed',
      error: 'tooLarge',
    })
    const readable = { ...everything, error: null }

    expect(getState(everything)).toBe('stuck')
    expect(getState(readable)).toBe('loading')
    expect(getState({ ...readable, written: 'one', isStale: true })).toBe('stale')
    expect(getState({ ...readable, written: 'one' })).toBe('saving')
    expect(getState({ ...readable, written: 'one', pendingWrite: null })).toBe('unsaved')
    expect(getState(tab())).toBe('clean')
  })
})

describe('a read answers with a body', () => {
  it('fills a loading tab and leaves it clean', () => {
    const next = transitionTab(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'one', at: 'a1' },
    })

    expect(next.tab.written).toBe('one')
    expect(next.tab.filePath).toBe('a1')
    expect(next.tab.shown).toBe('one')
    expect(getState(next.tab)).toBe('clean')
    expect(next.effects).toEqual([{ kind: 'replace', body: 'one' }])
  })

  it('replaces the document when the body differs from what is shown', () => {
    const next = transitionTab(tab(), {
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
    const next = transitionTab(tab(), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'one', at: 'a1' },
    })

    expect(next.effects).toEqual([])
    expect(next.tab).toEqual(tab())
  })

  it('is discarded while something is unsaved', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })
    const next = transitionTab(unsaved, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })

    expect(next.tab).toEqual(unsaved)
    expect(next.effects).toEqual([])
  })

  it('is discarded when it is older than the newest read issued', () => {
    const reading = tab({ reading: 3 })
    const next = transitionTab(reading, {
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
    const next = transitionTab(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    })

    expect(next.tab.written).toBe('')
    expect(next.tab.shown).toBe('')
    expect(getState(next.tab)).toBe('clean')

    const typed = transitionTab(next.tab, { kind: 'edited', body: 'again', at: 0 })
    expect(transitionTab(typed.tab, { kind: 'timerFired' }).effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'again', seen: null },
    ])
  })

  it('leaves the buffer alone out of a re-read, and says the note is not there', () => {
    // The note was removed or renamed under an open tab. What the person is
    // reading stays on the screen, and the tab stops writing it anywhere.
    const reading = tab({ reading: 2 })
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'missing' },
    })

    expect(next.tab.shown).toBe(reading.shown)
    expect(getState(next.tab)).toBe('gone')
    expect(next.effects).toEqual([])
  })

  it('leaves a tab standing on a live file alone where the read was stale', () => {
    // The note moved, the tab followed it and asked again. The read left behind
    // answers about the name the tab has left, and the file it is on is there.
    const reading = tab({ reading: 3 })
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'missing' },
    })

    expect(next).toEqual({ tab: reading, effects: [] })
    expect(getState(next.tab)).toBe('clean')
  })
})

describe('a read answers an error', () => {
  it('sticks a loading tab', () => {
    const next = transitionTab(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'error', error: 'notText' },
    })

    expect(next.tab.error).toBe('notText')
    expect(getState(next.tab)).toBe('stuck')
    expect(next.effects).toEqual([])
  })

  it('leaves a loading tab alone where the read was stale', () => {
    const reading = tab({ written: null, shown: '', reading: 2 })
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'error', error: 'notText' },
    })

    expect(next).toEqual({ tab: reading, effects: [] })
    expect(getState(next.tab)).toBe('loading')
  })

  it('strands nothing: a tab that never read its note is not mended by typing', () => {
    // Clearing the error here would take the tab back to loading, where the
    // interval does nothing, and what the person typed would never be written.
    const stuck = transitionTab(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'error', error: 'tooLarge' },
    }).tab

    const typed = transitionTab(stuck, { kind: 'edited', body: 'anything', at: 10 })

    expect(typed.tab.error).toBe('tooLarge')
    expect(getState(typed.tab)).toBe('stuck')
    expect(transitionTab(typed.tab, { kind: 'timerFired' }).effects).toEqual([])
    expect(transitionTab(typed.tab, { kind: 'close' }).effects).toEqual([
      { kind: 'reportError', error: 'tooLarge' },
      { kind: 'close' },
    ])
  })

  it('is discarded out of a re-read', () => {
    const reading = tab({ reading: 2 })
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'error', error: 'tooLarge' },
    })

    expect(next.tab).toEqual(reading)
    expect(next.effects).toEqual([])
  })
})

describe('the person types', () => {
  it('changes what is shown, marks when it began, and arms the interval', () => {
    const next = transitionTab(tab(), { kind: 'edited', body: 'onemore', at: 1000 })

    expect(next.tab.shown).toBe('onemore')
    expect(next.tab.since).toBe(1000)
    expect(getState(next.tab)).toBe('unsaved')
    expect(next.effects).toEqual([{ kind: 'startTimer', after: DEFAULT_WRITE_LIMITS.quiet }])
  })

  it('keeps when the first unwritten change was made', () => {
    const next = transitionTab(tab({ shown: 'on', since: 1000 }), {
      kind: 'edited',
      body: 'one',
      at: 1200,
    })

    expect(next.tab.since).toBe(1000)
  })

  it('puts the write off while a write is in the air', () => {
    const next = transitionTab(tab({ shown: 'one!', pendingWrite: 'one', since: 1000 }), {
      kind: 'edited',
      body: 'one!!',
      at: 1100,
    })

    expect(next.tab.shown).toBe('one!!')
    expect(getState(next.tab)).toBe('saving')
    expect(next.effects).toEqual([{ kind: 'startTimer', after: DEFAULT_WRITE_LIMITS.quiet }])
  })

  it('mends an error that is about the body', () => {
    const next = transitionTab(tab({ shown: 'huge', error: 'tooLarge', since: 10 }), {
      kind: 'edited',
      body: 'small',
      at: 20,
    })

    expect(next.tab.error).toBeNull()
    expect(getState(next.tab)).toBe('unsaved')
  })

  it('leaves an error that is a condition of the file', () => {
    const next = transitionTab(tab({ shown: 'edit', error: 'unreadable', since: 10 }), {
      kind: 'edited',
      body: 'edited',
      at: 20,
    })

    expect(next.tab.error).toBe('unreadable')
    expect(getState(next.tab)).toBe('stuck')
  })

  it('does nothing to a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })
    const next = transitionTab(loading, { kind: 'edited', body: 'early', at: 10 })

    expect(next.tab).toEqual(loading)
    expect(next.effects).toEqual([])
  })
})

describe('the interval fires', () => {
  it('begins a write of what is on screen when something is unsaved', () => {
    const next = transitionTab(tab({ shown: 'two', since: 10 }), { kind: 'timerFired' })

    expect(getState(next.tab)).toBe('saving')
    expect(next.tab.pendingWrite).toBe('two')
    expect(next.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', fingerprint: 'a1' } },
    ])
  })

  it('owes a write while one is in the air', () => {
    const next = transitionTab(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'timerFired',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
    expect(next.effects).toEqual([])
  })

  it('writes nothing when nothing changed', () => {
    const next = transitionTab(tab(), { kind: 'timerFired' })

    expect(next.tab).toEqual(tab())
    expect(next.effects).toEqual([])
  })

  it('writes nothing for a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })
    const next = transitionTab(loading, { kind: 'timerFired' })

    expect(next.tab).toEqual(loading)
    expect(next.effects).toEqual([])
  })

  it('writes nothing while the tab is stuck', () => {
    const stuck = tab({ shown: 'two', error: 'unreadable', since: 10 })
    const next = transitionTab(stuck, { kind: 'timerFired' })

    expect(next.tab).toEqual(stuck)
    expect(next.effects).toEqual([])
  })

  it('is armed no later than the bound after the first unwritten change', () => {
    // Typing without a pause: every keystroke puts the write off, and the bound
    // is what makes one happen.
    let held = tab()
    let after = DEFAULT_WRITE_LIMITS.quiet
    for (let at = 0; at <= DEFAULT_WRITE_LIMITS.bound; at += 100) {
      const next = transitionTab(held, { kind: 'edited', body: `${at}`, at })
      held = next.tab
      after = next.effects[0]?.kind === 'startTimer' ? next.effects[0].after : after
      expect(at + after).toBeLessThanOrEqual(DEFAULT_WRITE_LIMITS.bound)
    }

    expect(after).toBe(0)
    expect(getState(transitionTab(held, { kind: 'timerFired' }).tab)).toBe('saving')
  })
})

describe('a write answers ok', () => {
  it('takes the body it carried as written, and forgets when the change was made', () => {
    const next = transitionTab(tab({ shown: 'two', pendingWrite: 'two', since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok', at: 'a2' },
    })

    expect(next.tab.written).toBe('two')
    expect(next.tab.filePath).toBe('a2')
    expect(next.tab.since).toBeNull()
    expect(getState(next.tab)).toBe('clean')
    expect(next.effects).toEqual([])
  })

  it('leaves the tab unsaved when the person typed while it was in the air', () => {
    const next = transitionTab(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'written',
      answer: { kind: 'ok', at: 'a2' },
    })

    expect(next.tab.written).toBe('two')
    expect(isDirty(next.tab)).toBe(true)
    expect(getState(next.tab)).toBe('unsaved')
  })

  it('begins the write it owed', () => {
    const next = transitionTab(
      tab({ shown: 'three', pendingWrite: 'two', hasPendingWrite: true, since: 10 }),
      {
        kind: 'written',
        answer: { kind: 'ok', at: 'a2' },
      },
    )

    expect(getState(next.tab)).toBe('saving')
    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'three', seen: { prose: 'two', fingerprint: 'a2' } },
    ])
  })
})

describe('the fingerprint the file was read at', () => {
  it('is what the next write presents, taken from the write that answered', () => {
    // Two saves with no read between. The file is what this tab's own write left
    // behind, and the fingerprint that write answered with is what it carries.
    const first = transitionTab(transitionTab(tab(), { kind: 'edited', body: 'two', at: 0 }).tab, {
      kind: 'timerFired',
    })
    expect(first.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', fingerprint: 'a1' } },
    ])

    const written = transitionTab(first.tab, { kind: 'written', answer: { kind: 'ok', at: 'a2' } })
    const again = transitionTab(written.tab, { kind: 'edited', body: 'three', at: 100 })
    const second = transitionTab(again.tab, { kind: 'timerFired' })

    expect(second.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'three', seen: { prose: 'two', fingerprint: 'a2' } },
    ])
  })

  it('is nothing for a file that was not there, and that write compares nothing', () => {
    const made = transitionTab(tab({ written: null, shown: '' }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    })

    expect(made.tab.filePath).toBeNull()
  })
})

describe('a write answers changed', () => {
  it('overtakes the tab and leaves what was shown where it is', () => {
    const next = transitionTab(tab({ shown: 'mine', pendingWrite: 'mine', since: 10 }), {
      kind: 'written',
      answer: { kind: 'changed' },
    })

    expect(getState(next.tab)).toBe('stale')
    expect(next.tab.pendingWrite).toBeNull()
    expect(next.tab.shown).toBe('mine')
    expect(next.tab.written).toBe('one')
    expect(next.effects).toEqual([])
  })

  it('drops the write that was owed', () => {
    const next = transitionTab(
      tab({ shown: 'mine', pendingWrite: 'was', hasPendingWrite: true, since: 10 }),
      { kind: 'written', answer: { kind: 'changed' } },
    )

    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.effects).toEqual([])
  })

  it('arms nothing while the person types into a stale tab', () => {
    const next = transitionTab(createStaleTab(), { kind: 'edited', body: 'mine and more', at: 2000 })

    expect(next.tab.shown).toBe('mine and more')
    expect(getState(next.tab)).toBe('stale')
    expect(next.effects).toEqual([])
  })

  it('writes nothing when an interval armed before it fires', () => {
    const stopped = createStaleTab()
    const next = transitionTab(stopped, { kind: 'timerFired' })

    expect(next.tab).toEqual(stopped)
    expect(next.effects).toEqual([])
  })

  it('reads nothing when the vault changes under it', () => {
    const stopped = createStaleTab()

    expect(transitionTab(stopped, { kind: 'changed', paths: ['Note.md'], renamed: [] })).toEqual({
      tab: stopped,
      effects: [],
    })
    expect(transitionTab(stopped, { kind: 'changed', paths: [], renamed: [] })).toEqual({
      tab: stopped,
      effects: [],
    })
  })

  it('holds a tab that the person types back to what was written', () => {
    const stopped = createStaleTab()
    const back = transitionTab(stopped, { kind: 'edited', body: 'one', at: 2000 }).tab

    expect(isDirty(back)).toBe(false)
    expect(getState(back)).toBe('stale')
    expect(transitionTab(back, { kind: 'keepLocal' }).effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'one', seen: null },
    ])
  })

  it('discards an answer to a write it is no longer waiting on', () => {
    const stopped = createStaleTab()
    const next = transitionTab(stopped, {
      kind: 'written',
      answer: { kind: 'error', error: 'tooLarge' },
    })

    expect(next.tab).toEqual(stopped)
    expect(getState(next.tab)).toBe('stale')
  })
})

describe('the person keeps theirs', () => {
  it('writes what is shown, comparing nothing', () => {
    const next = transitionTab(createStaleTab(), { kind: 'keepLocal' })

    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'mine', seen: null }])
    expect(getState(next.tab)).toBe('saving')
    expect(next.tab.isStale).toBe(false)
  })

  it('takes what that write answers with, and the tab is clean', () => {
    const writing = transitionTab(createStaleTab(), { kind: 'keepLocal' }).tab
    const next = transitionTab(writing, { kind: 'written', answer: { kind: 'ok', at: 'a3' } })

    expect(next.tab.written).toBe('mine')
    expect(next.tab.filePath).toBe('a3')
    expect(getState(next.tab)).toBe('clean')
  })

  it('does nothing to a tab that was not stale', () => {
    const unsaved = tab({ shown: 'two', since: 10 })

    expect(transitionTab(unsaved, { kind: 'keepLocal' })).toEqual({ tab: unsaved, effects: [] })
  })
})

describe("the person takes the file's", () => {
  it('reads again, and the tab stays stale until that read lands', () => {
    const next = transitionTab(createStaleTab(), { kind: 'takeFile' })

    expect(next.tab.reading).toBe(2)
    expect(getState(next.tab)).toBe('stale')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('replaces the buffer with what the read answers, unsaved as the tab is', () => {
    const reading = transitionTab(createStaleTab(), { kind: 'takeFile' }).tab
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })

    expect(next.effects).toEqual([{ kind: 'replace', body: 'theirs' }])
    expect(next.tab.shown).toBe('theirs')
    expect(next.tab.written).toBe('theirs')
    expect(next.tab.filePath).toBe('a2')
    expect(getState(next.tab)).toBe('clean')
  })

  it('leaves the bound to start again with the next change', () => {
    const typing = transitionTab(createStaleTab(), { kind: 'edited', body: 'mine and more', at: 1000 }).tab
    const reading = transitionTab(typing, { kind: 'takeFile' }).tab
    const took = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    }).tab

    expect(took.since).toBeNull()
    expect(
      transitionTab(took, { kind: 'edited', body: 'theirs and mine', at: 9000 }).effects,
    ).toEqual([{ kind: 'startTimer', after: DEFAULT_WRITE_LIMITS.quiet }])
  })

  it('stays stale for a read that is out of date', () => {
    const reading = transitionTab(createStaleTab(), { kind: 'takeFile' }).tab
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'body', body: 'stale', at: 'a0' },
    })

    expect(next.tab).toEqual(reading)
    expect(getState(next.tab)).toBe('stale')
  })

  it('says the note is not there where the file is gone, since there is nothing to take', () => {
    const reading = transitionTab(createStaleTab(), { kind: 'takeFile' }).tab
    const next = transitionTab(reading, {
      kind: 'read',
      generation: 2,
      answer: { kind: 'missing' },
    })

    expect(next.tab.shown).toBe(reading.shown)
    expect(getState(next.tab)).toBe('gone')
  })

  it('does nothing to a tab that was not stale', () => {
    const clean = tab()

    expect(transitionTab(clean, { kind: 'takeFile' })).toEqual({ tab: clean, effects: [] })
  })
})

describe('a save is requested for', () => {
  it('writes what is unsaved now', () => {
    const next = transitionTab(tab({ shown: 'two', since: 10 }), { kind: 'save' })

    expect(next.effects).toEqual([
      { kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', fingerprint: 'a1' } },
    ])
    expect(getState(next.tab)).toBe('saving')
  })

  it('owes one while a write is in the air', () => {
    const next = transitionTab(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'save',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
    expect(next.effects).toEqual([])
  })

  it('writes nothing when nothing changed', () => {
    const clean = tab()

    expect(transitionTab(clean, { kind: 'save' })).toEqual({ tab: clean, effects: [] })
  })

  it('writes nothing for a tab that has not read its note', () => {
    const loading = tab({ written: null, shown: '' })

    expect(transitionTab(loading, { kind: 'save' })).toEqual({ tab: loading, effects: [] })
  })

  it('writes nothing while the tab is stuck', () => {
    const stuck = tab({ shown: 'two', error: 'unreadable', since: 10 })

    expect(transitionTab(stuck, { kind: 'save' })).toEqual({ tab: stuck, effects: [] })
  })

  it('writes nothing for a stale tab, which the person answers', () => {
    const stopped = createStaleTab()

    expect(transitionTab(stopped, { kind: 'save' })).toEqual({ tab: stopped, effects: [] })
  })
})

describe('a file about to be renamed or removed', () => {
  it('lets go of the interval it is waiting on', () => {
    const clean = tab()

    expect(transitionTab(clean, { kind: 'flush' })).toEqual({
      tab: clean,
      effects: [{ kind: 'cancelTimer' }],
    })
  })

  it('writes what is unsaved to the path it still remains at', () => {
    const next = transitionTab(tab({ shown: 'two', since: 10 }), { kind: 'flush' })

    expect(next.effects).toEqual([
      { kind: 'cancelTimer' },
      { kind: 'write', path: 'Note.md', body: 'two', seen: { prose: 'one', fingerprint: 'a1' } },
    ])
    expect(getState(next.tab)).toBe('saving')
  })

  it('owes a write for what is unsaved behind one already in the air', () => {
    const next = transitionTab(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'flush',
    })

    expect(next.tab.hasPendingWrite).toBe(true)
    expect(kinds(next.effects)).toEqual(['cancelTimer'])
  })

  it('owes nothing behind a write that carries everything typed', () => {
    const next = transitionTab(tab({ pendingWrite: 'one' }), { kind: 'flush' })

    expect(next.tab.hasPendingWrite).toBe(false)
    expect(kinds(next.effects)).toEqual(['cancelTimer'])
  })

  it('writes nothing for a stale tab, whose question is the person to answer', () => {
    const stopped = createStaleTab()
    const next = transitionTab(stopped, { kind: 'flush' })

    expect(next.tab).toEqual(stopped)
    expect(kinds(next.effects)).toEqual(['cancelTimer'])
  })

  it('writes nothing for a tab whose note is no longer there', () => {
    const missing = transitionTab(tab({ reading: 1 }), {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    }).tab
    const next = transitionTab(missing, { kind: 'flush' })

    expect(next.tab).toEqual(missing)
    expect(kinds(next.effects)).toEqual(['cancelTimer'])
  })

  it('leaves the tab where it is, so the note is followed wherever it went', () => {
    const flushed = transitionTab(tab(), { kind: 'flush' }).tab
    const next = transitionTab(flushed, {
      kind: 'changed',
      paths: [],
      renamed: [{ from: 'Note.md', to: 'Moved.md' }],
    })

    expect(next.tab.path).toBe('Moved.md')
  })
})

describe('a write answers an error', () => {
  it('sticks the tab, drops what was owed, and leaves the buffer editable', () => {
    const next = transitionTab(
      tab({ shown: 'two', pendingWrite: 'two', hasPendingWrite: true, since: 10 }),
      {
        kind: 'written',
        answer: { kind: 'error', error: 'tooLarge' },
      },
    )

    expect(getState(next.tab)).toBe('stuck')
    expect(next.tab.hasPendingWrite).toBe(false)
    expect(next.tab.shown).toBe('two')

    const typed = transitionTab(next.tab, { kind: 'edited', body: 'small', at: 20 })
    expect(typed.tab.shown).toBe('small')
  })
})

describe('the vault changes', () => {
  it('reads again when the change names this path and nothing is unsaved', () => {
    const next = transitionTab(tab({ reading: 1 }), {
      kind: 'changed',
      paths: ['Other.md', 'Note.md'],
      renamed: [],
    })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('reads again for a reload, which is empty at all', () => {
    const next = transitionTab(tab({ reading: 1 }), { kind: 'changed', paths: [], renamed: [] })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('reads again when a gone tab is changed on disk', () => {
    const gone = tab({ isDeleted: true, reading: 1 })
    const next = transitionTab(gone, {
      kind: 'changed',
      paths: ['Note.md'],
      renamed: [],
    })

    expect(next.tab.reading).toBe(2)
    expect(next.effects).toEqual([{ kind: 'read', path: 'Note.md', generation: 2 }])
  })

  it('does nothing to a tab with something unsaved, reload or not', () => {
    const unsaved = tab({ shown: 'mine', since: 10 })

    expect(transitionTab(unsaved, { kind: 'changed', paths: ['Note.md'], renamed: [] })).toEqual({
      tab: unsaved,
      effects: [],
    })
    expect(transitionTab(unsaved, { kind: 'changed', paths: [], renamed: [] })).toEqual({
      tab: unsaved,
      effects: [],
    })
  })

  it('does nothing while loading, saving or stuck', () => {
    const held: Tab[] = [
      tab({ written: null, shown: '' }),
      tab({ shown: 'two', pendingWrite: 'two', since: 10 }),
      tab({ error: 'unreadable' }),
    ]

    for (const one of held) {
      expect(transitionTab(one, { kind: 'changed', paths: [], renamed: [] })).toEqual({
        tab: one,
        effects: [],
      })
    }
  })

  it('does nothing when the change names other paths only', () => {
    const clean = tab()
    const next = transitionTab(clean, {
      kind: 'changed',
      paths: ['Somewhere/Else.md'],
      renamed: [],
    })

    expect(next.tab).toEqual(clean)
    expect(next.effects).toEqual([])
  })
})

describe('a close is requested for', () => {
  it('writes what is unsaved and holds the tab', () => {
    const next = transitionTab(tab({ shown: 'two', since: 10 }), { kind: 'close' })

    expect(kinds(next.effects)).toEqual(['write', 'keepOpen'])
    expect(next.effects[0]).toEqual({
      kind: 'write',
      path: 'Note.md',
      body: 'two',
      seen: { prose: 'one', fingerprint: 'a1' },
    })
    expect(getState(next.tab)).toBe('saving')
  })

  it('holds a tab with a write in the air, and owes it what was typed since', () => {
    const next = transitionTab(tab({ shown: 'three', pendingWrite: 'two', since: 10 }), {
      kind: 'close',
    })

    expect(next.effects).toEqual([{ kind: 'keepOpen' }])
    expect(next.tab.hasPendingWrite).toBe(true)
  })

  it('closes a tab with nothing unsaved', () => {
    const next = transitionTab(tab(), { kind: 'close' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('closes a tab that has not read its note, which owes nothing', () => {
    const next = transitionTab(tab({ written: null, shown: '' }), { kind: 'close' })

    expect(next.effects).toEqual([{ kind: 'close' }])
  })

  it('holds a stale tab, writing nothing and leaving the question standing', () => {
    const next = transitionTab(createStaleTab(), { kind: 'close' })

    expect(kinds(next.effects)).toEqual(['keepOpen'])
    expect(getState(next.tab)).toBe('stale')
    expect(next.tab.shown).toBe('mine')
  })

  it('says why a stuck tab cannot be written before it goes', () => {
    const next = transitionTab(tab({ shown: 'two', error: 'unreadable', since: 10 }), {
      kind: 'close',
    })

    expect(next.effects).toEqual([{ kind: 'reportError', error: 'unreadable' }, { kind: 'close' }])
  })
})

describe('a note that moved', () => {
  it('is followed to where it went', () => {
    const next = transitionTab(tab(), {
      kind: 'changed',
      paths: ['Note.md', 'Renamed.md'],
      renamed: [{ from: 'Note.md', to: 'Renamed.md' }],
    })
    expect(next.tab.path).toBe('Renamed.md')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Renamed.md', generation: 2 }])
  })

  it('is followed even where the tab has something unwritten, so its save lands where the note is', () => {
    const next = transitionTab(tab({ shown: 'mine', since: 10 }), {
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
    const next = transitionTab(tab(), {
      kind: 'changed',
      paths: ['Other.md'],
      renamed: [{ from: 'Other.md', to: 'Elsewhere.md' }],
    })
    expect(next.tab.path).toBe('Note.md')
    expect(kinds(next.effects)).toEqual([])
  })
})

describe('a note that is no longer there', () => {
  const applyMissingRead = (over: Partial<Tab> = {}) =>
    transitionTab(tab({ reading: 2, ...over }), {
      kind: 'read',
      generation: 2,
      answer: { kind: 'missing' },
    }).tab

  it('leaves what the person has on the screen', () => {
    expect(applyMissingRead({ shown: 'mine', since: 10 }).shown).toBe('mine')
  })

  it('is a state of its own, so the tab can say so', () => {
    expect(getState(applyMissingRead())).toBe('gone')
  })

  it('stops the unasked save, so nothing is written back without being requested', () => {
    const next = transitionTab(applyMissingRead({ shown: 'mine', since: 10 }), {
      kind: 'timerFired',
    })
    expect(kinds(next.effects)).toEqual([])
  })

  it('is not written by typing either', () => {
    const typed = transitionTab(applyMissingRead(), { kind: 'edited', body: 'more', at: 20 })
    expect(getState(typed.tab)).toBe('gone')
    expect(kinds(transitionTab(typed.tab, { kind: 'timerFired' }).effects)).toEqual([])
  })

  it('is made again at the name it had when the person says to keep it', () => {
    const next = transitionTab(applyMissingRead({ shown: 'mine', since: 10 }), {
      kind: 'keepLocal',
    })
    expect(next.effects).toEqual([{ kind: 'write', path: 'Note.md', body: 'mine', seen: null }])
  })

  it('is itself again once the note arrives back', () => {
    const back = transitionTab(applyMissingRead(), {
      kind: 'changed',
      paths: ['Note.md'],
      renamed: [],
    })
    const read = transitionTab(back.tab, {
      kind: 'read',
      generation: back.tab.reading,
      answer: { kind: 'body', body: 'theirs', at: 'a2' },
    })
    expect(getState(read.tab)).toBe('clean')
    expect(read.tab.shown).toBe('theirs')
  })
})

describe('the word a tab carries alongside its title', () => {
  it('is nothing while the note is being read', () => {
    expect(getMarkOf('loading')).toBeUndefined()
  })

  it('is nothing when the text on screen is the text of the file', () => {
    expect(getMarkOf('clean')).toBeUndefined()
  })

  it('is unsaved while there is something to write', () => {
    expect(getMarkOf('unsaved')).toBe('unsaved')
  })

  it('is unsaved while the write is on its way', () => {
    expect(getMarkOf('saving')).toBe('unsaved')
  })

  it('is stale while the file has moved past what the tab read', () => {
    expect(getMarkOf('stale')).toBe('stale')
  })

  it('is stuck when the note can be neither read nor written', () => {
    expect(getMarkOf('stuck')).toBe('stuck')
  })

  it('is a different word for each of the three things a tab carries', () => {
    const words: (string | undefined)[] = [
      getMarkOf('stuck'),
      getMarkOf('stale'),
      getMarkOf('unsaved'),
    ]
    expect(new Set(words).size).toBe(3)
  })
})

describe('a tab that has requested for nothing to be written', () => {
  it('asks for nothing when the first read finds no file', () => {
    const next = transitionTab(openTab('Note.md').tab, {
      kind: 'read',
      generation: 1,
      answer: { kind: 'missing' },
    })

    expect(next.effects).toEqual([])
  })
})

describe('a write that failed', () => {
  it('asks for nothing further', () => {
    const next = transitionTab(tab({ shown: 'two', pendingWrite: 'two', since: 10 }), {
      kind: 'written',
      answer: { kind: 'error', error: 'tooLarge' },
    })

    expect(next.effects).toEqual([])
  })
})

describe('a rename of the file under a tab', () => {
  it('reads the file at its new path, though the change names only the old one', () => {
    const next = transitionTab(tab({ reading: 1 }), {
      kind: 'changed',
      paths: ['Note.md'],
      renamed: [{ from: 'Note.md', to: 'Moved.md' }],
    })

    expect(next.tab.path).toBe('Moved.md')
    expect(next.effects).toEqual([{ kind: 'read', path: 'Moved.md', generation: 2 }])
  })
})
