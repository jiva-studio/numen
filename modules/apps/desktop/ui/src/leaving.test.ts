/**
 * The route by which quitting reaches a buffer only the page holds.
 *
 * The failure this guards against is silent: the window goes, the application
 * answers that everything landed, and the last seconds of typing are gone.
 */
import { describe, expect, it } from 'vitest'

import { editing } from './editing'
import { leaving, type Going, type Owed, type Question } from './leaving'
import type { Answered, Core } from './showing'

/** A vault that has been read and is doing nothing. */
const idle = {
  name: '',
  ready: true,
  failed: '',
  unwatched: '',
  chunks: 0n,
  embedded: 0n,
  reading: '',
  embedding: false,
  books: 0n,
  booksRead: 0n,
  learning: false,
  busy: false,
}

/** What the application says over the quit stream, when a test says it. */
function stream() {
  const said: { token: string; flush: boolean }[] = []
  let wake: (() => void) | null = null
  let over = false
  return {
    say(message: { token: string; flush: boolean }) {
      said.push(message)
      wake?.()
    },
    /** The stream drops. The next one a page opens is a stream again. */
    end() {
      over = true
      wake?.()
    },
    async *read() {
      for (;;) {
        while (said.length > 0) yield said.shift() as { token: string; flush: boolean }
        if (over) {
          over = false
          return
        }
        await new Promise<void>((woken) => {
          wake = woken
        })
      }
    },
  }
}

/** A core whose reads and writes a test drives, and whose quit it speaks for. */
function fake(quitting: () => AsyncIterable<{ token: string; flush: boolean }>) {
  const files = new Map<string, string>()
  const wrote: { path: string; body: string }[] = []
  const answered: { token: string; owed: Owed }[] = []
  /** Writes wait here until a test lets them through. */
  let held: (() => void) | null = null

  const core: Core & Going = {
    neighbourhood: async () => ({}) as never,
    opening: async () => null,
    state: async () => idle,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    quitting,
    flushed: async (token: string, owed: Owed = 'written') => {
      answered.push({ token, owed })
    },
    read: async (path): Promise<Answered> =>
      files.has(path)
        ? { body: files.get(path) ?? '', refusal: null }
        : { body: '', refusal: 'missing' },
    write: async (path, body): Promise<Answered> => {
      if (held) await new Promise<void>((through) => (held = through))
      wrote.push({ path, body })
      files.set(path, body)
      return { body: '', refusal: null }
    },
    create: async () => ({ path: '', refusal: null }),
    join: async () => null,
  }
  return {
    core,
    files,
    wrote,
    answered,
    hold: () => {
      held = () => {}
    },
    release: () => {
      const through = held
      held = null
      through?.()
    },
  }
}

/**
 * A tab whose text the file changed under, which a test answers for. The write
 * each way out would do is not this file's subject; what it records is which
 * way the person took.
 */
function conflicted(path: string) {
  const took: string[] = []
  let drop = () => {}
  const question: Question = {
    path,
    keep: async () => {
      took.push('keep ' + path)
      drop()
    },
    take: async () => {
      took.push('take ' + path)
      drop()
    },
  }
  return {
    question,
    took,
    raise: (raising: (one: Question) => () => void) => {
      drop = raising(question)
    },
    /** The question stops standing, however that came about. */
    answered: () => drop(),
  }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))
const settle = async () => {
  for (let i = 0; i < 20; i++) await nap()
}

describe('a page asked to write what it owes', () => {
  it('writes an unsaved tab and only then says it has', async () => {
    const said = stream()
    const at = fake(said.read)
    const notes = editing(at.core)
    const going = leaving(at.core)
    going.holds(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.typed('Note.md', 'what the person was in the middle of')

    // No interval has fired, so what was typed is in the page and nowhere else.
    expect(at.wrote).toEqual([])

    said.say({ token: '7', flush: true })
    await settle()

    expect(at.wrote).toEqual([{ path: 'Note.md', body: 'what the person was in the middle of' }])
    expect(at.answered).toEqual([{ token: '7', owed: 'written' }])
  })

  it('does not answer while the write it owes is still in the air', async () => {
    const said = stream()
    const at = fake(said.read)
    const notes = editing(at.core)
    const going = leaving(at.core)
    going.holds(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.typed('Note.md', 'held')
    at.hold()

    said.say({ token: '1', flush: true })
    await settle()

    expect(at.answered).toEqual([])

    at.release()
    await settle()

    expect(at.wrote).toEqual([{ path: 'Note.md', body: 'held' }])
    expect(at.answered).toEqual([{ token: '1', owed: 'written' }])
  })

  it('says nothing until the application asks', async () => {
    const said = stream()
    const at = fake(said.read)
    const notes = editing(at.core)
    const going = leaving(at.core)
    going.holds(notes.flush)
    void going.start()

    notes.open('Note.md')
    await settle()
    notes.typed('Note.md', 'still being written')

    // The stream opens by handing over the token, which asks for nothing.
    said.say({ token: '3', flush: false })
    await settle()

    expect(at.wrote).toEqual([])
    expect(at.answered).toEqual([])
  })

  it('answers for a window with nothing open', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core)
    going.holds(editing(at.core).flush)
    void going.start()

    said.say({ token: '0', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '0', owed: 'written' }])
  })
})

describe('a page holding text the file changed under', () => {
  it('says there are questions outstanding, and does not say it has written', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core)
    const note = conflicted('Note.md')
    note.raise(going.raise)
    void going.start()

    said.say({ token: '4', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '4', owed: 'asking' }])
    expect(going.questions.value.map((one) => one.path)).toEqual(['Note.md'])
  })

  it('says so before the writes it owes have landed', async () => {
    const said = stream()
    const at = fake(said.read)
    const notes = editing(at.core)
    const going = leaving(at.core)
    going.holds(notes.flush)
    const note = conflicted('Held.md')
    note.raise(going.raise)

    void going.start()
    notes.open('Other.md')
    await settle()
    notes.typed('Other.md', 'on its way')
    at.hold()

    said.say({ token: '5', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '5', owed: 'asking' }])

    at.release()
    await settle()

    // The write landed and the question still stands, so nothing has changed
    // about what the page owes.
    expect(at.wrote).toEqual([{ path: 'Other.md', body: 'on its way' }])
    expect(at.answered).toEqual([{ token: '5', owed: 'asking' }])
  })

  it('says it has written once every question is answered', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core)
    const first = conflicted('One.md')
    const second = conflicted('Two.md')
    first.raise(going.raise)
    second.raise(going.raise)
    void going.start()

    said.say({ token: '6', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '6', owed: 'asking' }])

    await going.questions.value.find((one) => one.path === 'One.md')?.keep()
    await settle()

    // One of the two is answered, so the window is still owed something.
    expect(at.answered.at(-1)).toEqual({ token: '6', owed: 'asking' })
    expect(going.questions.value.map((one) => one.path)).toEqual(['Two.md'])

    await going.questions.value.find((one) => one.path === 'Two.md')?.take()
    await settle()

    expect(at.answered.at(-1)).toEqual({ token: '6', owed: 'written' })
    expect(going.questions.value).toEqual([])
    expect(first.took).toEqual(['keep One.md'])
    expect(second.took).toEqual(['take Two.md'])
  })

  it('leaves a question the person put off standing, and stops drawing it', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core, async () => {})
    conflicted('Later.md').raise(going.raise)
    void going.start()

    said.say({ token: '8', flush: true })
    await settle()

    going.questions.value[0]?.later()
    await settle()

    expect(going.questions.value).toEqual([])
    // Nothing was answered for it, so the window is still owed it.
    expect(at.answered.at(-1)).toEqual({ token: '8', owed: 'asking' })

    // And it is still owed it the next time the window is asked for.
    said.end()
    await settle()
    said.say({ token: '9', flush: true })
    await settle()

    expect(at.answered.at(-1)).toEqual({ token: '9', owed: 'asking' })
  })

  it('says nothing under a token the stream took with it', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core, async () => {})
    const note = conflicted('Note.md')
    note.raise(going.raise)
    void going.start()

    said.say({ token: '2', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '2', owed: 'asking' }])

    said.end()
    await settle()
    note.answered()
    await settle()

    // Nothing answers to that token now, and the page has not been asked again.
    expect(at.answered).toEqual([{ token: '2', owed: 'asking' }])

    said.say({ token: '3', flush: true })
    await settle()

    expect(at.answered.at(-1)).toEqual({ token: '3', owed: 'written' })
  })

  it('raises what stands again under the token it is asked under next', async () => {
    const said = stream()
    const at = fake(said.read)
    const going = leaving(at.core, async () => {})
    conflicted('Note.md').raise(going.raise)
    void going.start()

    said.say({ token: '9', flush: true })
    await settle()

    expect(at.answered).toEqual([{ token: '9', owed: 'asking' }])

    // The stream drops and the page listens again under a new token.
    said.end()
    await settle()
    said.say({ token: '10', flush: true })
    await settle()

    expect(at.answered.at(-1)).toEqual({ token: '10', owed: 'asking' })
  })
})
