/**
 * What the quit waits for, tested without a window.
 *
 * A prompt that outlives the conflict that raised it holds the window open on a
 * note the user has already resolved; a conflict that never raises a prompt
 * lets the window close with the text still unsaved.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { raiseConflicts, type Notes } from './conflicts'
import { asFailure, asValue } from '@numen/core-client'
import { openNotes } from '@/entities/note'
import { answerNotes, type NoteCore as NotesCore } from '@/testing/noteCore'
import type { Conflict } from './flush'

/** The words this test puts its notes in. A screen has more; these are enough. */
type Word = 'clean' | 'unsaved' | 'stale' | 'gone'

const notes = () => {
  const states = ref<Record<string, Word>>({})
  const calls: string[] = []
  const store: Notes = {
    getOpenIds: () => Object.keys(states.value),
    getOpenNote: (id) => ({ state: states.value[id] ?? 'clean' }),
    keep: (id) => calls.push(`keep ${id}`),
    take: (id) => calls.push(`take ${id}`),
  }
  return {
    store,
    calls,
    setState: (id: string, state: Word) => {
      states.value = { ...states.value, [id]: state }
    },
  }
}

const quit = () => {
  const raised = new Map<string, Conflict>()
  return {
    going: {
      raise: (one: Conflict) => {
        raised.set(one.note, one)
        return () => raised.delete(one.note)
      },
    },
    paths: () => [...raised.keys()],
    answer: async (id: string, how: 'keep' | 'take') => raised.get(id)?.[how](),
  }
}

const window = () => {
  const store = notes()
  const going = quit()
  const stop = raiseConflicts(store.store, going.going)
  return { ...store, ...going, stop }
}

describe('a note the file moved past', () => {
  it('is a question the quit waits for when stale', async () => {
    const one = window()

    one.setState('Note.md', 'stale')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })

  it('answers with what the person chose', async () => {
    const one = window()
    one.setState('Note.md', 'stale')
    await nextTick()

    await one.answer('Note.md', 'keep')

    expect(one.calls).toEqual(['keep Note.md'])
  })

  it('answers with what the person took', async () => {
    const one = window()
    one.setState('Note.md', 'stale')
    await nextTick()

    await one.answer('Note.md', 'take')

    expect(one.calls).toEqual(['take Note.md'])
  })

  it('is raised again when it goes stale again after it was dropped', async () => {
    const one = window()
    one.setState('Note.md', 'stale')
    await nextTick()
    one.setState('Note.md', 'clean')
    await nextTick()

    one.setState('Note.md', 'stale')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })

  it('is dropped once the note is no longer stale', async () => {
    const one = window()
    one.setState('Note.md', 'stale')
    await nextTick()

    one.setState('Note.md', 'clean')
    await nextTick()

    expect(one.paths()).toEqual([])
  })

  it('is raised once while active, however often the notes change', async () => {
    const one = window()
    one.setState('Note.md', 'stale')
    await nextTick()
    one.setState('Other.md', 'unsaved')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })

  it('drops raised conflicts and stops watching when stopped', async () => {
    const store = notes()
    let dropCalls = 0
    const raised = new Set<string>()
    const stop = raiseConflicts(store.store, {
      raise: (one) => {
        raised.add(one.note)
        return () => {
          dropCalls += 1
          raised.delete(one.note)
        }
      },
    })

    store.setState('Note.md', 'stale')
    await nextTick()
    expect([...raised]).toEqual(['Note.md'])

    stop()
    expect([...raised]).toEqual([])
    expect(dropCalls).toBe(1)

    stop()
    expect(dropCalls).toBe(1)

    store.setState('Other.md', 'stale')
    await nextTick()
    expect([...raised]).toEqual([])
  })
})

describe('a note that is written', () => {
  it('is no question at all', async () => {
    const one = window()

    one.setState('Note.md', 'clean')
    one.setState('Other.md', 'unsaved')
    await nextTick()

    expect(one.paths()).toEqual([])
  })
})

describe('a window of real notes', () => {
  /** A core holding the files a test puts in it. */
  const createCore = (files: Map<string, string>): NotesCore => ({
    read: async (path) => asValue({ body: files.get(path) ?? '', at: `1 1 at:${files.get(path)}` }),
    write: async (path, body, observed) => {
      if (
        observed &&
        files.get(path) !== observed.prose &&
        `1 1 at:${files.get(path)}` !== observed.stat
      ) {
        return asFailure('changed' as const)
      }
      files.set(path, body)
      return asValue({ body: '', at: `1 1 at:${body}` })
    },
  })

  const flushMicrotasks = async () => {
    for (let i = 0; i < 4; i++) await new Promise((resolve) => setTimeout(resolve, 0))
    await nextTick()
  }

  /** A stale note A beside a clean note B, with the conflicts watched. */
  const createStaleConflict = async () => {
    const files = new Map([
      ['A.md', 'a'],
      ['B.md', 'b'],
    ])
    answerNotes(createCore(files))
    const notes = openNotes({ limits: { quiet: 1000, bound: 5000 } })
    const going = quit()
    let raisedCount = 0
    raiseConflicts(notes, {
      raise: (one) => (raisedCount++, going.going.raise(one)),
    })
    notes.open('A.md')
    notes.open('B.md')
    await flushMicrotasks()

    notes.setBody('A.md', 'mine')
    files.set('A.md', 'theirs')
    notes.save('A.md')
    await flushMicrotasks()
    return { notes, going, raised: () => raisedCount }
  }

  it('raises the conflict of the stale note and keeps it while another is typed into', async () => {
    const { notes, going, raised } = await createStaleConflict()
    expect(going.paths()).toEqual(['A.md'])

    notes.setBody('B.md', 'b and more')
    await flushMicrotasks()

    expect(going.paths()).toEqual(['A.md'])
    expect(raised()).toBe(1)
  })

  it('keeps the conflict while the stale note itself is typed into, and drops it once answered', async () => {
    const { notes, going, raised } = await createStaleConflict()

    notes.setBody('A.md', 'mine and more')
    await flushMicrotasks()
    expect(going.paths()).toEqual(['A.md'])
    expect(raised()).toBe(1)

    await going.answer('A.md', 'keep')
    await flushMicrotasks()
    expect(going.paths()).toEqual([])
  })
})
