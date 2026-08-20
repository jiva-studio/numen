/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { showing, type Core, type Task } from './showing'
import type { Neighbourhood } from './plex'

const answer = (path: string): Neighbourhood =>
  ({ focus: { path, title: path, identifier: '' }, related: [] }) as unknown as Neighbourhood

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

describe('a plex the window is following', () => {
  it('opens on the note the vault opens with', async () => {
    const window = showing(fake(), async () => window.close())
    await window.start()

    const plex = window.plex()
    await nap()

    expect(plex.here.value).toBe('Opening.md')
  })

  it('opens where the person is looking, when another one is open', async () => {
    const window = showing(fake(), async () => window.close())
    const first = window.plex()
    await nap()
    await first.go('Here.md')

    const second = window.plex()
    await nap()

    expect(second.here.value).toBe('Here.md')
  })

  it('asks the vault once at launch, and not once for the plex as well', async () => {
    let openings = 0
    const core = fake({
      opening: async () => {
        openings++
        return { path: 'Opening.md' }
      },
    })
    const window = showing(core, async () => window.close())

    const plex = window.plex()
    await window.start()
    await nap()

    expect(plex.here.value).toBe('Opening.md')
    expect(openings).toBe(1)
  })

  it('says what it could not show, where the window says everything else', async () => {
    // A neighbourhood of a note the index does not hold: a focus with no path.
    const core = fake({
      neighbourhood: async () =>
        ({ focus: { path: '', title: '' }, related: [] }) as unknown as Neighbourhood,
    })
    const window = showing(core, async () => window.close())

    await window.plex().go('Gone.md')

    expect(window.warning.value).toContain('Gone.md')
  })

  it('says nothing again once the note it could not show comes back', async () => {
    let holds = false
    const core = fake({
      neighbourhood: async (path) =>
        holds
          ? answer(path)
          : ({ focus: { path: '', title: '' }, related: [] }) as unknown as Neighbourhood,
    })
    const window = showing(core, async () => window.close())
    const plex = window.plex()

    await plex.go('Note.md')
    expect(window.warning.value).toContain('Note.md')

    holds = true
    await plex.go('Note.md')

    expect(window.warning.value).toBe('')
  })

  it('says the trouble of the plex the person is in, and not that of another', async () => {
    const core = fake({
      neighbourhood: async (path) =>
        path === 'Gone.md'
          ? (({ focus: { path: '', title: '' }, related: [] }) as unknown as Neighbourhood)
          : answer(path),
    })
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()

    one.looking()
    await one.go('Gone.md')
    await two.go('Two.md')

    // Two is standing where it asked to be, and one is not: the window says so
    // while the person is in one, and holds its tongue while they are in two.
    expect(window.warning.value).toContain('Gone.md')
    two.looking()
    expect(window.warning.value).toBe('')
    one.looking()
    expect(window.warning.value).toContain('Gone.md')
  })

  it('says nothing for a plex whose tab has closed', async () => {
    const core = fake({
      neighbourhood: async () =>
        ({ focus: { path: '', title: '' }, related: [] }) as unknown as Neighbourhood,
    })
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()

    const going = two.go('Gone.md')
    two.close()
    await going

    expect(window.warning.value).toBe('')
    expect(one.here.value).toBe('')
  })

  it('hands the front to the plex the person was in before, when one closes', async () => {
    const window = showing(fake(), async () => window.close())
    const one = window.plex()
    const two = window.plex()
    const three = window.plex()
    await one.go('One.md')
    await two.go('Two.md')
    await three.go('Three.md')

    two.looking()
    three.looking()
    three.close()

    expect(window.looking.value).toBe('Two.md')
  })

  it('says nothing about an unreachable core beyond the failure itself', async () => {
    const core = fake({
      opening: async () => {
        throw new Error('[unavailable] connection refused')
      },
      state: async () => {
        throw new Error('[unavailable] connection refused')
      },
    })
    const window = showing(core, async () => window.close())

    window.plex()
    await window.start()
    await nap()

    expect(window.failure.value).toContain('connection refused')
    expect(window.warning.value).toBe('')
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
    expect(window.warning.value).toContain('connection lost')
  })

  it('asks for the picture again for a change to any note', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const window = showing(core, async () => window.close())
    const plex = window.plex()
    await nap()

    await plex.go('Here.md')
    core.asked.length = 0
    await window.follow()

    expect(core.asked).toContain('Here.md')
  })

  it('asks every plex, since each is standing somewhere of its own', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()
    await nap()

    await one.go('One.md')
    await two.go('Two.md')
    core.asked.length = 0
    await window.follow()

    expect(core.asked).toContain('One.md')
    expect(core.asked).toContain('Two.md')
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
    const window = showing(core, async () => window.close())
    window.plex()
    window.plex()
    window.plex()
    await nap()

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
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()
    await one.go('One.md')
    await two.go('Two.md')
    openings = 0

    await window.follow()

    expect(openings).toBe(0)
  })

  it('gives a plex standing nowhere the note an empty vault has just gained', async () => {
    let note: { path: string } | null = null
    const core = fake({
      opening: async () => note,
      changes: async function* () {
        note = { path: 'First.md' }
        yield { paths: ['First.md'], reload: false, renamed: [] }
      },
    })
    const window = showing(core, async () => window.close())
    const plex = window.plex()
    await nap()
    expect(plex.here.value).toBe('')

    await window.follow()

    expect(plex.here.value).toBe('First.md')
    expect(window.holds.value).toBe(true)
  })

  it('stops asking for a plex whose tab has closed', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()
    await nap()

    await one.go('One.md')
    await two.go('Two.md')
    two.close()
    core.asked.length = 0
    await window.follow()

    expect(core.asked).toContain('One.md')
    expect(core.asked).not.toContain('Two.md')
  })
})

describe('a note asked for from outside the window', () => {
  it('is put in front of the plex the person is looking at', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md' }
      },
    })
    const window = showing(core, async () => window.close())
    const one = window.plex()
    const two = window.plex()
    await one.go('Opening.md')
    await nap()

    two.looking()
    await window.watch()

    expect(two.here.value).toBe('Wanted.md')
    expect(one.here.value).toBe('Opening.md')
  })

  it('opens a plex on it when the window has none, rather than being dropped', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md' }
      },
    })
    const opened: string[] = []
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      (path) => {
        opened.push(path)
        void window.plex(path)
      },
    )

    await window.watch()
    await nap()

    expect(opened).toStrictEqual(['Wanted.md'])
    expect(window.looking.value).toBe('Wanted.md')
  })
})

describe('a place inside a source asked for from outside the window', () => {
  /** A window that records the documents it was asked to open, and where. */
  const watching = (core: Core) => {
    const opened: string[] = []
    const window = showing(
      core,
      async () => window.close(),
      undefined,
      undefined,
      undefined,
      (path, start, length) => opened.push(`${path} ${start} ${length}`),
    )
    return { window, opened }
  }

  it('opens the document it stands in, and leaves the plex where it is', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'library/mahabharata.epub', start: 40_512, length: 31 }
      },
    })
    const { window, opened } = watching(core)
    const plex = window.plex()
    await plex.go('Opening.md')
    await nap()

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 31'])
    expect(plex.here.value).toBe('Opening.md')
  })

  it('travels the plex for a focus that names no run of a source', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md', start: 0, length: 0 }
      },
    })
    const { window, opened } = watching(core)
    const plex = window.plex()
    await plex.go('Opening.md')
    await nap()

    await window.watch()

    expect(opened).toStrictEqual([])
    expect(plex.here.value).toBe('Wanted.md')
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
    expect(window.warning.value).toBe('')

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
