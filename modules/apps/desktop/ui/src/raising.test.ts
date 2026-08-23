/**
 * What the quit is told to wait for, asked without a window.
 *
 * A question that outlives what raised it holds the window open on a note the
 * person has already answered; one that is never raised lets the window go
 * with the text still unwritten.
 */
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { raising, type Notes } from './raising'
import type { Question } from './leaving'

/** Notes in the states the test puts them in. */
const notes = () => {
  const states = ref<Record<string, string>>({})
  const said: string[] = []
  const store: Notes = {
    all: () => Object.keys(states.value),
    shown: (path) => ({ state: states.value[path] ?? 'clean' }),
    keep: (path) => said.push(`keep ${path}`),
    take: (path) => said.push(`take ${path}`),
  }
  return {
    store,
    said,
    stands: (path: string, state: string) => {
      states.value = { ...states.value, [path]: state }
    },
  }
}

/** A quit that keeps every question standing until it is dropped. */
const quit = () => {
  const standing = new Map<string, Question>()
  return {
    going: {
      raise: (one: Question) => {
        standing.set(one.path, one)
        return () => standing.delete(one.path)
      },
    },
    paths: () => [...standing.keys()],
    answer: async (path: string, how: 'keep' | 'take') => standing.get(path)?.[how](),
  }
}

const window = () => {
  const store = notes()
  const going = quit()
  raising(store.store, going.going)
  return { ...store, ...going }
}

describe('a note the file moved past', () => {
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
