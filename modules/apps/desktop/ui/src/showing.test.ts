/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { showing } from './showing'
import type { Core, Run, Task } from './core'
import type { Neighbourhood } from './core'
import { NeighbourhoodSchema } from './plex/picture'

const answer = (path: string): Neighbourhood =>
  create(NeighbourhoodSchema, { focus: { path, title: path, identifier: '' }, related: [] })

/** A neighbourhood of a note the index does not hold: a focus with no path. */
const nothing = (): Neighbourhood =>
  create(NeighbourhoodSchema, { focus: { path: '', title: '', identifier: '' }, related: [] })

/** A stream that stays open, so a loop waiting on it is not the one under test. */
const held = () => new Promise<never>(() => {})

const settled = {
  name: 'Vault',
  path: '/vaults/Physics',
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
    rename: async (path, title) => ({
      path,
      title,
      frontmatter: false,
      moved: null,
      refusal: null,
      changed: false,
    }),
    remove: async () => ({ trashed: '', dangling: [], refusal: null }),
    list: async () => [],
    move: async () => ({ moved: null, refusal: null }),
    makeFolder: async () => null,
    syncing: async () => true,
    choosesSyncing: async () => null,
    // eslint-disable-next-line require-yield
    quitting: async function* () {},
    flushed: async () => {},
    ...over,
  }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))

/**
 * What the window makes of what it is told: that the vault changed, and that a
 * note is wanted in front of the person. What each tab does about either is
 * asked where that tab is.
 */
const heard = (core: Core, reads?: (path: string, runs: readonly Run[]) => void) => {
  const changed: string[] = []
  const wanted: string[] = []
  const showed = showing(
    core,
    async () => showed.close(),
    async (paths, renamed) => {
      changed.push([...paths, ...(renamed ?? []).map((one) => `${one.from} → ${one.to}`)].join(' '))
    },
    undefined,
    async (path) => {
      wanted.push(path)
    },
    reads,
  )
  return { window: showed, changed, wanted }
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

  it('says what changed, and waits for whatever is drawn from it', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Somewhere/Else.md'], reload: false, renamed: [] }
      },
    })
    const one = heard(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual(['Somewhere/Else.md'])
  })

  it('says a reload names nothing at all, so everything reads again', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: ['Note.md'], reload: true, renamed: [] }
      },
    })
    const one = heard(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual([''])
  })

  it('says where a note went, for whatever is showing it to follow', async () => {
    const core = fake({
      changes: async function* () {
        yield { paths: [], reload: false, renamed: [{ from: 'Note.md', to: 'Renamed.md' }] }
      },
    })
    const one = heard(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual(['Note.md → Renamed.md'])
  })

  it('asks where the vault opens again when that is the note that moved', async () => {
    let opens = 'Opening.md'
    const core = fake({
      opening: async () => ({ path: opens }),
      changes: async function* () {
        opens = 'Renamed.md'
        yield { paths: [], reload: false, renamed: [{ from: 'Opening.md', to: 'Renamed.md' }] }
      },
    })
    const one = heard(core)
    await one.window.first()

    await one.window.follow()

    expect(one.window.opening.value).toBe('Renamed.md')
  })

  it('leaves where the vault opens alone when another note moved', async () => {
    let asked = 0
    const core = fake({
      opening: async () => {
        asked++
        return { path: 'Opening.md' }
      },
      changes: async function* () {
        yield { paths: [], reload: false, renamed: [{ from: 'Other.md', to: 'Renamed.md' }] }
      },
    })
    const one = heard(core)
    await one.window.first()

    await one.window.follow()

    expect(one.window.opening.value).toBe('Opening.md')
    expect(asked).toBe(1)
  })
})

/**
 * A vault is swapped from outside this window, and the reload that says so
 * looks exactly like the one that says the whole vault changed on disk. The
 * folder it stands at is what tells them apart.
 */
describe('another vault under this window', () => {
  /** What the window did about a reload: drew the page again, or read again. */
  const swapping = (core: Core) => {
    const drawn: string[] = []
    const window = showing(
      core,
      async () => {},
      async (paths) => void drawn.push(`told ${paths.join(' ')}`),
      undefined,
      undefined,
      undefined,
      () => void drawn.push('reloads'),
    )
    return { window, drawn }
  }

  /**
   * A vault that reloads, with every other stream of the window left open. The
   * notes named are changes the stream carries ahead of the reload.
   */
  const reloading = (state: Core['state'], ahead: readonly string[] = []) =>
    fake({
      state,
      changes: async function* () {
        for (const path of ahead) yield { paths: [path], reload: false, renamed: [] }
        yield { paths: [], reload: true, renamed: [] }
        await held()
      },
      focus: async function* () {
        await held()
      },
      editing: async function* () {
        await held()
      },
    })

  it('draws the page again where the reload stands at another folder', async () => {
    /** The folder the vault stands at, which the swap moves between reads. */
    const folders = ['/vaults/Physics', '/vaults/Heat']
    const one = swapping(reloading(async () => ({ ...settled, path: folders.shift() ?? '' })))

    await one.window.start()
    await nap()
    await nap()

    expect(one.drawn.at(-1)).toBe('reloads')

    one.window.close()
  })

  /**
   * The vault is read again for every change it reports, and a change of the
   * vault that arrived reaches the window before the reload does. The folder
   * the page was drawn on is what the reload is measured against.
   */
  it('draws the page again where the vault that arrived was read first', async () => {
    /** The folder the vault stands at: the first read is the page's own. */
    const folders = ['/vaults/Physics']
    const one = swapping(
      reloading(async () => ({ ...settled, path: folders.shift() ?? '/vaults/Heat' }), ['Heat.md']),
    )

    await one.window.start()
    await nap()
    await nap()
    await nap()

    expect(one.drawn).toContain('told Heat.md')
    expect(one.drawn.at(-1)).toBe('reloads')

    one.window.close()
  })

  it('reads the vault again where the reload stands where it stood', async () => {
    const one = swapping(reloading(async () => settled))

    await one.window.start()
    await nap()
    await nap()

    expect(one.drawn).not.toContain('reloads')
    expect(one.drawn.at(-1)).toBe('told ')

    one.window.close()
  })
})

describe('a note asked for from outside the window', () => {
  it('is put in front of the person, wherever the window draws it', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md' }
      },
    })
    const one = heard(core)

    await one.window.watch()

    expect(one.wanted).toStrictEqual(['Wanted.md'])
  })
})

describe('a place inside a source asked for from outside the window', () => {
  /** A window that records the documents it was asked to open, and where. */
  const watching = (core: Core) => {
    const opened: string[] = []
    const one = heard(core, (path, runs) =>
      opened.push(`${path} ${runs.map((run) => `${run.start} ${run.length}`).join(' ')}`),
    )
    return { window: one.window, opened, wanted: one.wanted }
  }

  it('opens the document it stands in, and asks for no note at all', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'library/mahabharata.epub', start: 40_512, length: 31 }
      },
    })
    const { window, opened, wanted } = watching(core)

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 31'])
    expect(wanted).toStrictEqual([])
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

  it('asks for the note itself where the focus names no run of a source', async () => {
    const core = fake({
      focus: async function* () {
        yield { path: 'Wanted.md', start: 0, length: 0 }
      },
    })
    const { window, opened, wanted } = watching(core)

    await window.watch()

    expect(opened).toStrictEqual([])
    expect(wanted).toStrictEqual(['Wanted.md'])
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
