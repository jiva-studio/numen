/**
 * The route by which quitting reaches a buffer only the page holds.
 *
 * The failure this guards against is silent: the window goes, the application
 * answers that everything landed, and the last seconds of typing are gone.
 */
import { describe, expect, it } from 'vitest'

import { editing } from './editing'
import { leaving } from './leaving'
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
    end() {
      over = true
      wake?.()
    },
    async *read() {
      for (;;) {
        while (said.length > 0) yield said.shift() as { token: string; flush: boolean }
        if (over) return
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
  const answered: string[] = []
  /** Writes wait here until a test lets them through. */
  let held: (() => void) | null = null

  const core: Core = {
    neighbourhood: async () => ({}) as never,
    opening: async () => null,
    state: async () => idle,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    quitting,
    flushed: async (token) => {
      answered.push(token)
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
    expect(at.answered).toEqual(['7'])
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
    expect(at.answered).toEqual(['1'])
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

    expect(at.answered).toEqual(['0'])
  })
})
