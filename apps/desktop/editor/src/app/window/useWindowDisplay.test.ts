/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { describe, expect, it, vi } from 'vitest'
import { asValue } from '@numen/wire'
import { useWindowDisplay } from './useWindowDisplay'
import type { Core } from '@/app/ports/core'
import type { Task } from '@/shared/notices/task'
import type { Span } from '@/shared/span'
import type { Neighbourhood } from '@/entities/note'

const answer = (path: string): Neighbourhood => ({
  focus: { path, title: path },
  focusType: 'note',
  related: [],
})

/** A stream that stays open, so a loop waiting on it is not the one under test. */
const waitForever = () => new Promise<never>(vi.fn())

const resolved = {
  id: '01JQVAULTPHYSICS0000000000',
  name: 'Vault',
  path: '/vaults/Physics',
  scan: { isReady: true, error: '', unwatchedPath: '' },
  coverage: { chunkCount: 0n, embeddedCount: 0n, isEmbedding: false },
  agentUnreachable: '',
}

/** A core that answers whatever it is told to, and records what it was asked. */
function fake(overrides: Partial<Core> = {}): Core & { asked: string[] } {
  const request: string[] = []
  return {
    asked: request,
    neighbourhood: async (path) => {
      request.push(path)
      return answer(path)
    },
    headings: async () => new Map(),
    fileKinds: async () => new Map(),
    resolve: async () => new Map(),
    getInitialOpenPath: async () => ({ path: 'Opening.md' }),
    state: async () => resolved,
    agentUnreachable: async () => '',
    watchVaultChanges: async function* () {},
    watchFocus: async function* () {},
    writeOpenTabs: vi.fn(() => Promise.resolve()),
    watchEdits: async function* () {},
    watchTasks: async function* () {
      await waitForever()
    },
    read: async () => asValue({ body: '' }),
    write: async () => asValue({ body: '' }),
    create: async () => ({ ok: true, value: { path: '' } }),
    join: async () => null,
    removeLink: async () => null,
    replaceLinks: async () => null,
    rename: async (path, title) => asValue({ path, title, hasFrontmatter: false, moved: null }),
    remove: async () => asValue({ trashed: '', dangling: [] }),
    list: async () => [],
    move: async () => ({ ok: true, value: null }),
    createFolder: async () => null,
    getSyncEnabled: async () => true,
    setSyncEnabled: async () => null,
    getSettings: async () => ({ written: '{}', path: '/numen.json', models: [] }),
    updateSettings: vi.fn(() => Promise.resolve()),
    writePluginSecret: vi.fn(() => Promise.resolve()),
    deletePluginSecret: vi.fn(() => Promise.resolve()),
    retryPlugin: vi.fn(() => Promise.resolve('')),
    choosePluginPath: vi.fn(() => Promise.resolve({ path: '', isChosen: false, problem: '' })),
    getPluginData: vi.fn(() => Promise.resolve(0)),
    deletePluginData: vi.fn(() => Promise.resolve()),
    getSettingsFile: async () => ({ written: '{}', path: '/numen.json' }),
    saveSettingsFile: async () => ({ isChanged: false }),
    watchQuit: async function* () {},
    reportFlush: vi.fn(() => Promise.resolve()),
    detachTab: vi.fn(() => Promise.resolve()),
    ...overrides,
  }
}

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

/**
 * What the window makes of what it is told: that the vault changed, and that a
 * note is wanted in front of the person. What each tab does about either is
 * asked where that tab is.
 */
const createDisplay = (core: Core, openFileAt?: (path: string, spans: readonly Span[]) => void) => {
  const changed: string[] = []
  const travelled: string[] = []
  const showed = useWindowDisplay(core, {
    wait: async () => showed.close(),
    onVaultChanged: async (paths, renamed) => {
      changed.push([...paths, ...(renamed ?? []).map((one) => `${one.from} → ${one.to}`)].join(' '))
    },
    travelTo: async (path) => {
      travelled.push(path)
    },
    ...(openFileAt ? { openFileAt } : {}),
  })
  return { window: showed, changed, travelled }
}

describe('the stream of changes', () => {
  it('is taken up again when it ends', async () => {
    let streams = 0
    const core = fake({
      watchVaultChanges: async function* () {
        streams++
        yield { paths: ['Note.md'], shouldReload: false, renamed: [] }
      },
    })
    const waits: number[] = []
    const window = useWindowDisplay(core, {
      wait: async (ms) => {
        waits.push(ms)
        if (streams >= 3) window.close()
      },
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(3)
    expect(waits.length).toBeGreaterThanOrEqual(2)
  })

  it('is taken up again when it fails, and says what happened', async () => {
    let streams = 0
    const core = fake({
      watchVaultChanges: function* () {
        streams++
        throw new Error('connection lost')
      } as unknown as Core['watchVaultChanges'],
    })
    const window = useWindowDisplay(core, {
      wait: async () => {
        if (streams >= 2) window.close()
      },
    })

    await window.follow()

    expect(streams).toBeGreaterThanOrEqual(2)
    expect(window.lost.value).toContain('lost touch with numen')
  })

  it('says what changed, and waits for whatever is drawn from it', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: ['Somewhere/Else.md'], shouldReload: false, renamed: [] }
      },
    })
    const one = createDisplay(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual(['Somewhere/Else.md'])
  })

  it('says a reload is empty at all, so everything reads again', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: ['Note.md'], shouldReload: true, renamed: [] }
      },
    })
    const one = createDisplay(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual([''])
  })

  it('says where a note went, for whatever is showing it to follow', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: [], shouldReload: false, renamed: [{ from: 'Note.md', to: 'Renamed.md' }] }
      },
    })
    const one = createDisplay(core)

    await one.window.follow()

    expect(one.changed).toStrictEqual(['Note.md → Renamed.md'])
  })

  it('asks where the vault opens again when that is the note that moved', async () => {
    let opens = 'Opening.md'
    const core = fake({
      getInitialOpenPath: async () => ({ path: opens }),
      watchVaultChanges: async function* () {
        opens = 'Renamed.md'
        yield {
          paths: [],
          shouldReload: false,
          renamed: [{ from: 'Opening.md', to: 'Renamed.md' }],
        }
      },
    })
    const one = createDisplay(core)
    await one.window.readInitialNote()

    await one.window.follow()

    expect(one.window.opening.value).toBe('Renamed.md')
  })

  it('leaves where the vault opens alone when another note moved', async () => {
    let request = 0
    const core = fake({
      getInitialOpenPath: async () => {
        request++
        return { path: 'Opening.md' }
      },
      watchVaultChanges: async function* () {
        yield { paths: [], shouldReload: false, renamed: [{ from: 'Other.md', to: 'Renamed.md' }] }
      },
    })
    const one = createDisplay(core)
    await one.window.readInitialNote()

    await one.window.follow()

    expect(one.window.opening.value).toBe('Opening.md')
    expect(request).toBe(1)
  })
})

/**
 * A vault is swapped from outside this window, and the reload that says so
 * looks exactly like the one that says the whole vault changed on disk. The
 */
describe('another vault under this window', () => {
  /** What the window did about a reload: drew the page again, or read again. */
  const createWindow = (core: Core) => {
    const renderedElement: string[] = []
    const window = useWindowDisplay(core, {
      wait: vi.fn(() => Promise.resolve()),
      onVaultChanged: (paths) => {
        renderedElement.push(`changed ${paths.join(' ')}`)
        return Promise.resolve()
      },
      reload: () => {
        renderedElement.push('reloaded')
      },
    })
    return { window, renderEvents: renderedElement }
  }

  /**
   * A vault that reloads, with every other stream of the window left open. The
   * notes named are changes the stream carries ahead of the reload.
   */
  const createReloadingCore = (state: Core['state'], ahead: readonly string[] = []) =>
    fake({
      state,
      watchVaultChanges: async function* () {
        for (const path of ahead) yield { paths: [path], shouldReload: false, renamed: [] }
        yield { paths: [], shouldReload: true, renamed: [] }
        await waitForever()
      },
      watchFocus: async function* () {
        await waitForever()
      },
      watchEdits: async function* () {
        await waitForever()
      },
    })

  it('draws the page again where the reload remains at another folder', async () => {
    const folders = ['/vaults/Physics', '/vaults/Heat']
    const one = createWindow(
      createReloadingCore(async () => ({ ...resolved, path: folders.shift() ?? '' })),
    )

    await one.window.start()
    await flushPromises()
    await flushPromises()

    expect(one.renderEvents.at(-1)).toBe('reloaded')

    one.window.close()
  })

  /**
   * The vault is read again for every change it reports, and a change of the
   * vault that arrived reaches the window before the reload does. The folder
   * the page was drawn on is what the reload is measured against.
   */
  it('draws the page again where the vault that arrived was read first', async () => {
    const folders = ['/vaults/Physics']
    const one = createWindow(
      createReloadingCore(
        async () => ({ ...resolved, path: folders.shift() ?? '/vaults/Heat' }),
        ['Heat.md'],
      ),
    )

    await one.window.start()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    expect(one.renderEvents).toContain('changed Heat.md')
    expect(one.renderEvents.at(-1)).toBe('reloaded')

    one.window.close()
  })

  it('reads the vault again where the vault cannot be requested at its current position', async () => {
    let reads = 0
    const one = createWindow(
      createReloadingCore(async () => {
        reads += 1
        if (reads > 1) throw new Error('connection lost')
        return resolved
      }),
    )

    await one.window.start()
    await flushPromises()
    await flushPromises()

    expect(one.renderEvents).not.toContain('reloaded')

    one.window.close()
  })

  it('reads the vault again where the reload remains where it stood', async () => {
    const one = createWindow(createReloadingCore(async () => resolved))

    await one.window.start()
    await flushPromises()
    await flushPromises()

    expect(one.renderEvents).not.toContain('reloaded')
    expect(one.renderEvents.at(-1)).toBe('changed ')

    one.window.close()
  })
})

describe('a note requested for from outside the window', () => {
  it('is put in front of the person, wherever the window draws it', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield { path: 'Wanted.md', spans: [] }
      },
    })
    const one = createDisplay(core)

    await one.window.watch()

    expect(one.travelled).toStrictEqual(['Wanted.md'])
  })
})

describe('a place inside a source requested for from outside the window', () => {
  /** A window that records the documents it was asked to open, and where. */
  const createRecordingWindow = (core: Core) => {
    const openedWindows: string[] = []
    const one = createDisplay(core, (path, spans) =>
      openedWindows.push(`${path} ${spans.map((one) => `${one.from} ${one.to}`).join(' ')}`),
    )
    return { window: one.window, opened: openedWindows, travelled: one.travelled }
  }

  it('opens the document it remains in, and asks for no note at all', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield { path: 'library/mahabharata.epub', spans: [{ from: 40_512, to: 40_543 }] }
      },
    })
    const { window, opened, travelled } = createRecordingWindow(core)

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 40543'])
    expect(travelled).toStrictEqual([])
  })

  it('opens the document at every place the focus names, the first of them first', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield {
          path: 'library/mahabharata.epub',
          spans: [
            { from: 40_512, to: 40_543 },
            { from: 41_000, to: 41_020 },
            // A span reaching nowhere is no place, and is not lit.
            { from: 42_000, to: 42_000 },
          ],
        }
      },
    })
    const { window, opened } = createRecordingWindow(core)

    await window.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 40543 41000 41020'])
  })

  it('asks for the note itself where the focus names no run of a source', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield { path: 'Wanted.md', spans: [{ from: 0, to: 0 }] }
      },
    })
    const { window, opened, travelled } = createRecordingWindow(core)

    await window.watch()

    expect(opened).toStrictEqual([])
    expect(travelled).toStrictEqual(['Wanted.md'])
  })
})

describe('a vault that could not be read', () => {
  it('stops the waiting and is not called empty', async () => {
    const core = fake({
      getInitialOpenPath: async () => null,
      state: async () => ({
        ...resolved,
        scan: { isReady: false, error: 'permission denied', unwatchedPath: '' },
      }),
    })
    const window = useWindowDisplay(core, { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.isIndexing.value).toBe(false)
    expect(window.error.value).toBe('permission denied')
    expect(window.hasNote.value).toBe(false)
  })
})

describe('a vault with a note in it', () => {
  it('says it holds one, which is not what an unreadable vault says', async () => {
    const window = useWindowDisplay(fake(), { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.hasNote.value).toBe(true)
    expect(window.error.value).toBe('')
  })
})

describe('a vault that is not being followed', () => {
  it('says so and not looking up to date', async () => {
    const core = fake({
      state: async () => ({
        ...resolved,
        scan: { ...resolved.scan, unwatchedPath: 'too many watches' },
      }),
    })
    const window = useWindowDisplay(core, { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.unwatched.value).toBe('too many watches')
  })
})

describe('chunks with nothing to embed them', () => {
  it('says so, since the vault is searched by its words from now on', async () => {
    const core = fake({
      state: async () => ({
        ...resolved,
        coverage: { chunkCount: 4823n, embeddedCount: 0n, isEmbedding: false },
      }),
    })
    const window = useWindowDisplay(core, { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.chunks.value).toBe(4823)
    expect(window.embedded.value).toBe(0)
    expect(window.isEmbedding.value).toBe(false)
  })
})

describe('what the application is doing', () => {
  const readText = (count: number): Task => ({
    id: 'reading:library/scan.pdf',
    label: 'Recognising a scan',
    about: 'library/scan.pdf',
    completedCount: count,
    total: 400,
    counting: 'items',
    error: '',
    isUserStarted: true,
    pluginId: '',
    pluginName: '',
  })

  it('is what the stream last returned, whole', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        await waitForever()
      },
      watchFocus: async function* () {
        await waitForever()
      },
      watchEdits: async function* () {
        await waitForever()
      },
      watchTasks: async function* () {
        yield [readText(16)]
        yield [readText(32)]
        await waitForever()
      },
    })
    const window = useWindowDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()

    expect(window.tasks.value.map((one) => one.completedCount)).toEqual([32])

    window.close()
  })

  it('takes the stream up again, and says nothing about having lost it', async () => {
    let openedWindows = 0
    const core = fake({
      watchVaultChanges: async function* () {
        await waitForever()
      },
      watchFocus: async function* () {
        await waitForever()
      },
      watchEdits: async function* () {
        await waitForever()
      },
      watchTasks: async function* () {
        openedWindows++
        if (openedWindows === 1) throw new Error('the stream dropped')
        yield [readText(48)]
        await waitForever()
      },
    })
    // The clock is the test's, so the wait between one stream and the next is
    // not a second of it.
    const window = useWindowDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()
    await flushPromises()

    expect(openedWindows).toBeGreaterThan(1)
    expect(window.tasks.value.map((one) => one.completedCount)).toEqual([48])
    // A stream taken up again is not a stream that was lost.
    expect(window.lost.value).toBe('')

    window.close()
  })

  it('asks what the vault holds again once the work is over', async () => {
    // Cutting a library moves the counts with no file changing, so the moment
    // the list empties is the moment they are worth asking for.
    let asks = 0
    const core = fake({
      watchVaultChanges: async function* () {
        await waitForever()
      },
      watchFocus: async function* () {
        await waitForever()
      },
      watchEdits: async function* () {
        await waitForever()
      },
      state: async () => {
        asks++
        return {
          ...resolved,
          coverage: { ...resolved.coverage, chunkCount: BigInt(asks * 1000) },
        }
      },
      watchTasks: async function* () {
        yield [readText(16)]
        yield []
        await waitForever()
      },
    })
    const window = useWindowDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()

    expect(asks).toBeGreaterThan(1)
    expect(window.chunks.value).toBe(asks * 1000)

    window.close()
  })
})

describe('a window before it has requested the vault anything', () => {
  it('says nothing yet, and is waiting for the vault to be read', () => {
    const window = useWindowDisplay(fake())

    expect(window.name.value).toBe('')
    expect(window.isIndexing.value).toBe(true)
    expect(window.failure.value).toBe('')
    expect(window.lost.value).toBe('')
    expect(window.error.value).toBe('')
    expect(window.unwatched.value).toBe('')
    expect(window.unreachable.value).toBe('')
    expect(window.hasNote.value).toBe(false)
    expect(window.opening.value).toBe('')
    expect(window.isEmbedding.value).toBe(false)
    expect(window.tasks.value).toStrictEqual([])
  })
})

describe('the note the vault opens with', () => {
  it('is none for a vault that holds no note', async () => {
    const window = useWindowDisplay(fake({ getInitialOpenPath: async () => null }))

    expect(await window.readInitialNote()).toBe('')
    expect(window.opening.value).toBe('')
    expect(window.hasNote.value).toBe(false)
  })
})

describe('the first note shown', () => {
  it('stops the waiting and notifies everything showing the vault to read it whole', async () => {
    const one = createDisplay(fake())

    await one.window.start()
    await flushPromises()

    expect(one.window.isIndexing.value).toBe(false)
    expect(one.changed).toStrictEqual([''])
  })

  it('says what changed in a note while it is being made', async () => {
    const edit = {
      change: 'c1',
      path: 'Note.md',
      span: { from: 0, to: 4 },
      text: 'Leaf',
      isComplete: false,
    }
    const response: unknown[] = []
    const core = fake({
      watchEdits: async function* () {
        yield edit
      },
    })
    const window = useWindowDisplay(core, {
      wait: waitForever,
      reportChange: (one) => {
        response.push(one)
      },
    })

    await window.start()
    await flushPromises()
    window.close()

    expect(response).toStrictEqual([edit])
  })

  it('is not shown by a window closed while the vault was being requested', async () => {
    const changed: unknown[] = []
    const core = fake({
      getInitialOpenPath: async () => {
        window.close()
        return { path: 'Opening.md' }
      },
    })
    const window = useWindowDisplay(core, {
      onVaultChanged: (paths) => {
        changed.push(paths)
      },
    })

    await window.start()
    await flushPromises()

    expect(window.isIndexing.value).toBe(true)
    expect(changed).toStrictEqual([])
  })

  it('is waited for while the vault holds none and is still being read', async () => {
    const changed: unknown[] = []
    const core = fake({
      getInitialOpenPath: async () => null,
      state: async () => ({ ...resolved, scan: { ...resolved.scan, isReady: false } }),
    })
    const waits: number[] = []
    const window = useWindowDisplay(core, {
      wait: async (ms) => {
        waits.push(ms)
        window.close()
      },
      onVaultChanged: (paths) => {
        changed.push(paths)
      },
    })

    await window.start()
    await flushPromises()

    expect(window.isIndexing.value).toBe(true)
    expect(changed).toStrictEqual([])
    expect(waits).toStrictEqual([100])
  })
})

describe('a closed window', () => {
  it('lets go of every stream it listens to', async () => {
    const signals: AbortSignal[] = []
    const core = fake({
      watchTasks: async function* (signal) {
        signals.push(signal)
        await waitForever()
      },
    })
    const window = useWindowDisplay(core, { wait: waitForever })

    await window.start()
    await flushPromises()
    window.close()

    expect(signals).toHaveLength(1)
    expect(signals[0]?.aborted).toBe(true)
  })
})

describe('a vault that cannot be reached at all', () => {
  it('says so, and stops the waiting', async () => {
    const window = useWindowDisplay(
      fake({
        state: async () => {
          throw new Error('connection failed')
        },
      }),
    )

    await window.start()

    expect(window.failure.value).not.toBe('')
    expect(window.isIndexing.value).toBe(false)
  })
})

describe('a vault still being read', () => {
  it('is requested again a tenth of a second later', async () => {
    vi.useFakeTimers()
    let reads = 0
    const window = useWindowDisplay(
      fake({
        getInitialOpenPath: async () => null,
        state: async () => {
          reads += 1
          return { ...resolved, scan: { ...resolved.scan, isReady: false } }
        },
      }),
    )

    void window.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(reads).toBe(1)
    await vi.advanceTimersByTimeAsync(100)
    window.close()
    vi.useRealTimers()

    expect(reads).toBe(2)
  })
})
