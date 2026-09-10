/**
 * What the quit is told to wait for, asked without a window.
 *
 * A question that outlives what raised it holds the window open on a note the
 * person has already answered; one that is never raised lets the window go
 * with the text still unwritten.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { raiseConflicts, type Notes } from './conflicts'
import type { Conflict } from './flushing'

/** The words this test puts its notes in. A screen has more; these are enough. */
type Word = 'clean' | 'unsaved' | 'stale' | 'overtaken' | 'gone'

/** Notes in the states the test puts them in, each under its own identity. */
const notes = () => {
  const states = ref<Record<string, Word>>({})
  const said: string[] = []
  const store: Notes = {
    all: () => Object.keys(states.value),
    shown: (id) => ({ state: states.value[id] ?? 'clean' }),
    keep: (id) => said.push(`keep ${id}`),
    take: (id) => said.push(`take ${id}`),
  }
  return {
    store,
    said,
    stands: (id: string, state: Word) => {
      states.value = { ...states.value, [id]: state }
    },
  }
}

/** A quit that keeps every conflict standing until it is dropped. */
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
  raiseConflicts(store.store, going.going)
  return { ...store, ...going }
}

describe('a note the file moved past', () => {
  it('is a question the quit waits for when stale', async () => {
    const one = window()

    one.stands('Note.md', 'stale')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })

  it('is a question the quit waits for', async () => {
    const one = window()

    one.stands('Note.md', 'overtaken')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })

  it('answers with what the person chose', async () => {
    const one = window()
    one.stands('Note.md', 'overtaken')
    await nextTick()

    await one.answer('Note.md', 'keep')

    expect(one.said).toEqual(['keep Note.md'])
  })

  it('is dropped once the note is no longer overtaken', async () => {
    const one = window()
    one.stands('Note.md', 'overtaken')
    await nextTick()

    one.stands('Note.md', 'clean')
    await nextTick()

    expect(one.paths()).toEqual([])
  })

  it('is raised once while it stands, however often the notes change', async () => {
    const one = window()
    one.stands('Note.md', 'overtaken')
    await nextTick()
    one.stands('Other.md', 'unsaved')
    await nextTick()

    expect(one.paths()).toEqual(['Note.md'])
  })
})

describe('a note that is written', () => {
  it('is no question at all', async () => {
    const one = window()

    one.stands('Note.md', 'clean')
    one.stands('Other.md', 'unsaved')
    await nextTick()

    expect(one.paths()).toEqual([])
  })
})
