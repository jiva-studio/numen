/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { showing, type Core, type Plexes, type Task } from './showing'
import { NeighbourhoodSchema, type Neighbourhood } from './plex'

const answer = (path: string): Neighbourhood =>
  create(NeighbourhoodSchema, { focus: { path, title: path, identifier: '' }, related: [] })

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = (): Neighbourhood =>
  create(NeighbourhoodSchema, { focus: { path: '', title: '', identifier: '' }, related: [] })

/** A stream that stays open, so a loop waiting on it is not the one under test. */
const held = () => new Promise<never>(() => {})

const settled = {
  name: 'Vault',
  ready: true,
  failed: '',
  unwatched: '',
  unreachable: '',
  chunks: 0n,
  embedded: 0n,
  embedding: false,
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
    tasks: async function* () {
      await held()
    },
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

/**
 * The plexes of a window, as far as following the vault reads them. What one
 * of them does when it is asked again is asked of the plexes themselves.
 */
const plexed = (nowhere = false) => {
  const asked: string[] = []
  const travelled: string[] = []
  const port: Plexes = {
    nowhere: () => nowhere,
    again: async () => {
      asked.push('again')
    },
    travel: async (path) => {
      travelled.push(path)
    },
  }
  return { port, asked, travelled }
}

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
    expect(window.lost.value).toContain('connection lost')
  })

  it('asks the plexes for their pictures again for a change to any note', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const plexes = plexed()
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      undefined,
      plexes.port,
    )

    await window.follow()

    expect(plexes.asked).toStrictEqual(['again'])
  })

  it('asks where the vault opens once, however many plexes stand nowhere', async () => {
    let openings = 0
    const core = fake({
      opening: async () => {
        openings++
        return null
      },
      changes: async function* () {
        yield { paths: ['Note.md'], reload: false, renamed: [] }
      },
    })
    const plexes = plexed(true)
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      undefined,
      plexes.port,
    )

    await window.follow()

    expect(openings).toBe(1)
  })

  it('does not ask where the vault opens while every plex is standing somewhere', async () => {
    let openings = 0
    const core = fake({
      opening: async () => {
        openings++
        return { path: 'Opening.md' }
      },
      changes: async function* () {
        yield { paths: ['Note.md'], reload: false, renamed: [] }
      },
    })
    const plexes = plexed(false)
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      undefined,
      plexes.port,
    )

    await window.follow()

    expect(openings).toBe(0)
  })
})

describe('a note asked for from outside the window', () => {
  it('is put in front of the plexes, which travel to it', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md' }
      },
    })
    const plexes = plexed()
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      undefined,
      plexes.port,
    )

    await window.watch()

    expect(plexes.travelled).toStrictEqual(['Wanted.md'])
  })
})

describe('a place inside a source asked for from outside the window', () => {
  /** A window that records the documents it was asked to open, and where. */
  const watching = (core: Core) => {
    const opened: string[] = []
    const plexes = plexed()
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      (path, runs) =>
        opened.push(`${path} ${runs.map((one) => `${one.start} ${one.length}`).join(' ')}`),
      plexes.port,
    )
    return { window, opened, travelled: plexes.travelled }
  }

  it('opens the document it stands in, and leaves the plexes where they are', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'library/mahabharata.epub', start: 40_512, length: 31 }
      },
    })
    const { window, opened, travelled } = watching(core)

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 31'])
    expect(travelled).toStrictEqual([])
  })

  it('opens the document at every place the focus names, the first of them first', async () => {
    const core = fake({
      focus: async function* () {
        yield {
          path: 'library/mahabharata.epub',
          start: 40_512,
          length: 31,
          also: [
            { start: 41_000, length: 20 },
            // A place of no length is no place, and is not lit.
            { start: 42_000, length: 0 },
          ],
        }
      },
    })
    const { window, opened } = watching(core)

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 31 41000 20'])
  })

  it('travels the plexes for a focus that names no run of a source', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md', start: 0, length: 0 }
      },
    })
    const { window, opened, travelled } = watching(core)

    await window.watch()

    expect(opened).toStrictEqual([])
    expect(travelled).toStrictEqual(['Wanted.md'])
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
    expect(window.holds.value).toBe(false)
  })
})

describe('a vault with a note in it', () => {
  it('says it holds one, which is not what an unreadable vault says', async () => {
    const window = showing(fake(), async () => window.close())

    await window.start()
    await nap()

    expect(window.holds.value).toBe(true)
    expect(window.trouble.value).toBe('')
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

describe('chunks with nothing to embed them', () => {
  it('says so, since the vault is searched by its words from now on', async () => {
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

describe('what the application is doing', () => {
  const reading = (done: number): Task => ({
    id: 'reading:library/scan.pdf',
    doing: 'Reading a scan',
    about: 'library/scan.pdf',
    done,
    total: 400,
    counting: 'things',
    failed: '',
    asked: true,
  })

  it('is what the stream last said, whole', async () => {
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
      tasks: async function* () {
        yield [reading(16)]
        yield [reading(32)]
        await held()
      },
    })
    const window = showing(core, async () => {})

    await window.start()
    await nap()

    expect(window.tasks.value.map((one) => one.done)).toEqual([32])

    window.close()
  })

  it('takes the stream up again, and says nothing about having lost it', async () => {
    let opened = 0
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
      tasks: async function* () {
        opened++
        if (opened === 1) throw new Error('the stream dropped')
        yield [reading(48)]
        await held()
      },
    })
    // The clock is the test's, so the wait between one stream and the next is
    // not a second of it.
    const window = showing(core, async () => {})

    await window.start()
    await nap()
    await nap()

    expect(opened).toBeGreaterThan(1)
    expect(window.tasks.value.map((one) => one.done)).toEqual([48])
    // A stream taken up again is not a stream that was lost.
    expect(window.lost.value).toBe('')

    window.close()
  })

  it('asks what the vault holds again once the work is over', async () => {
    // Cutting a library moves the counts with no file changing, so the moment
    // the list empties is the moment they are worth asking for.
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
        return { ...settled, chunks: BigInt(asks * 1000), embedding: false }
      },
      tasks: async function* () {
        yield [reading(16)]
        yield []
        await held()
      },
    })
    const window = showing(core, async () => {})

    await window.start()
    await nap()

    expect(asks).toBeGreaterThan(1)
    expect(window.chunks.value).toBe(asks * 1000)

    window.close()
  })
})
