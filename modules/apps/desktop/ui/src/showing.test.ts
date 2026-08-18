/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { showing, type Core } from './showing'
import type { Neighbourhood } from './plex'

const answer = (path: string): Neighbourhood =>
  ({ focus: { path, title: path, identifier: '' }, related: [] }) as unknown as Neighbourhood

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = () =>
  ({ focus: { path: '', title: '', identifier: '' }, related: [] }) as unknown as Neighbourhood

const settled = {
  name: 'Vault',
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

/** A core that answers whatever it is told to, and records what it was asked. */
function fake(over: Partial<Core> = {}): Core & { asked: string[] } {
  const asked: string[] = []
  return {
    asked,
    neighbourhood: async (path) => {
      asked.push(path)
      return answer(path)
    },
    opening: async () => ({ path: 'Opening.md' }),
    state: async () => settled,
    // eslint-disable-next-line require-yield
    changes: async function* () {},
    // eslint-disable-next-line require-yield
    focus: async function* () {},
    // eslint-disable-next-line require-yield
    editing: async function* () {},
    read: async () => ({ body: '', refusal: null }),
    write: async () => ({ body: '', refusal: null }),
    create: async () => ({ path: '', refusal: null }),
    join: async () => null,
    // eslint-disable-next-line require-yield
    quitting: async function* () {},
    flushed: async () => {},
    ...over,
  }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))

describe('two questions in flight', () => {
  it('keeps the answer to the last one asked, however they come back', async () => {
    const delays: Record<string, number> = { Slow: 30, Fast: 0 }
    const core = fake({
      neighbourhood: async (path) => {
        await new Promise((wake) => setTimeout(wake, delays[path] ?? 0))
        return answer(path)
      },
    })
    const window = showing(core)

    const slow = window.go('Slow')
    const fast = window.go('Fast')
    await Promise.all([slow, fast])

    expect(window.here.value).toBe('Fast')
    expect(window.neighbourhood.value?.focus?.path).toBe('Fast')
  })
})

describe('the note in focus goes away', () => {
  it('says so, keeps what it is showing, and can come back to it', async () => {
    let holds = true
    const core = fake({
      neighbourhood: async (path) => (holds ? answer(path) : nothing()),
    })
    const window = showing(core)

    await window.go('Note.md')
    expect(window.here.value).toBe('Note.md')

    holds = false
    await window.go('Note.md')
    expect(window.notice.value).toContain('Note.md')
    // The path it asked about is kept, which is the only way back to it.
    expect(window.here.value).toBe('Note.md')
    expect(window.neighbourhood.value?.focus?.path).toBe('Note.md')

    holds = true
    await window.go('Note.md')
    expect(window.notice.value).toBe('')
  })
})

describe('the stream of changes', () => {
  it('is taken up again when it ends', async () => {
    let streams = 0
    const core = fake({
      changes: async function* () {
        streams++
        yield { paths: ['Note.md'], reload: false, renamed: [] }
      },
    })
    const waits: number[] = []
    const window = showing(core, async (ms) => {
      waits.push(ms)
      if (streams >= 3) window.close()
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(3)
    expect(waits.length).toBeGreaterThanOrEqual(2)
  })

  it('is taken up again when it fails, and says what happened', async () => {
    let streams = 0
    const core = fake({
      changes: function* () {
        streams++
        throw new Error('connection lost')
      } as unknown as Core['changes'],
    })
    const window = showing(core, async () => {
      if (streams >= 2) window.close()
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(2)
    expect(window.notice.value).toContain('connection lost')
  })

  it('asks for the picture again for a change to any note', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const window = showing(core, async () => window.close())

    await window.go('Here.md')
    core.asked.length = 0
    await window.follow()

    expect(core.asked).toContain('Here.md')
  })
})

describe('a vault that could not be read', () => {
  it('stops the waiting and is not called empty', async () => {
    const core = fake({
      opening: async () => null,
      state: async () => ({ ...settled, ready: false, failed: 'permission denied' }),
    })
    const window = showing(core, async () => window.close())

    await window.start()
    await nap()

    expect(window.indexing.value).toBe(false)
    expect(window.trouble.value).toBe('permission denied')
  })
})

describe('a vault that is not being followed', () => {
  it('says so rather than looking up to date', async () => {
    const core = fake({ state: async () => ({ ...settled, unwatched: 'too many watches' }) })
    const window = showing(core, async () => window.close())

    await window.start()
    await nap()

    expect(window.unwatched.value).toBe('too many watches')
  })
})

/**
 * A stream that stays open, which is what a real one does.
 *
 * The stream stays open, so the interval under test is the only one being
 * waited.
 */
const held = () => new Promise<never>(() => {})

describe('chunks with nothing to embed them', () => {
  it('is not busy, so a count of none is not shown as work', async () => {
    const core = fake({
      state: async () => ({ ...settled, chunks: 4823n, embedded: 0n, embedding: false }),
    })
    const window = showing(core, async () => window.close())

    await window.start()
    await nap()

    expect(window.chunks.value).toBe(4823)
    expect(window.embedded.value).toBe(0)
    expect(window.embedding.value).toBe(false)
  })
})

describe('a vault reading itself', () => {
  it('asks again on its own, since none of that work touches a file', async () => {
    let asks = 0
    const core = fake({
      changes: async function* () {
        await held()
      },
      focus: async function* () {
        await held()
      },
      editing: async function* () {
        await held()
      },
      state: async () => {
        asks++
        // Cutting begins after the notes are read, so at the moment the window
        // opens there is nothing to count and the vault is the only one that
        // knows more is coming.
        return {
          ...settled,
          busy: asks < 4,
          chunks: BigInt(asks * 100),
          embedded: 0n,
          embedding: true,
        }
      },
    })
    const window = showing(core, async () => {
      if (asks > 6) window.close()
    })

    await window.start()
    await nap()

    // Asked again with no change reported and no note touched.
    expect(asks).toBeGreaterThanOrEqual(4)
    // And stopped once the vault said it was done.
    expect(asks).toBeLessThanOrEqual(6)
    expect(window.chunks.value).toBeGreaterThan(0)
  })

  it('measures how fast the count moves over the interval it waited', async () => {
    let asks = 0
    let clock = 0
    const core = fake({
      changes: async function* () {
        await held()
      },
      focus: async function* () {
        await held()
      },
      editing: async function* () {
        await held()
      },
      state: async () => {
        asks++
        return {
          ...settled,
          busy: asks < 5,
          chunks: 1000n,
          embedded: BigInt(asks * 20),
          embedding: true,
          learning: true,
        }
      },
    })
    const window = showing(
      core,
      async (ms) => {
        clock += ms
      },
      () => clock,
    )

    await window.start()
    await nap()

    // Twenty more every two seconds is ten a second.
    expect(window.rate.value).toBeCloseTo(10, 5)
  })

  it('starts the rate again when the phase changes, since it counts another thing', async () => {
    let asks = 0
    let clock = 0
    const core = fake({
      changes: async function* () {
        await held()
      },
      focus: async function* () {
        await held()
      },
      editing: async function* () {
        await held()
      },
      state: async () => {
        asks++
        if (asks < 4) {
          // Books, and forty of them.
          return { ...settled, busy: true, books: 40n, booksRead: BigInt(asks * 10), embedding: true }
        }
        // Now chunks, and ninety thousand already carry a vector. The count
        // jumps forward by three orders of magnitude.
        return {
          ...settled,
          busy: asks < 5,
          books: 40n,
          booksRead: 40n,
          chunks: 100000n,
          embedded: 90000n,
          embedding: true,
          learning: true,
        }
      },
    })
    const window = showing(
      core,
      async (ms) => {
        clock += ms
      },
      () => clock,
    )

    await window.start()
    await nap()

    expect(window.learning.value).toBe(true)
    // One reading of a phase is a count. Two are a rate.
    expect(window.rate.value).toBe(0)
  })
})
