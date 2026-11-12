/**
 * The rules that decide whether the window keeps up with the vault.
 *
 * Every one of these has a failure that looks like nothing at all: a window
 * showing something stale, with no error and no way back.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TASK_KIND, TASK_UNIT } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import { useWindowDisplay } from './useWindowDisplay'
import type { Task } from '@/entities/window'
import type { TextRange } from '@/shared/lib/textRange'
import type { NoteEdit } from '@/entities/note'

async function* emptyStream(): AsyncGenerator<never> {
  yield* []
}

const waitForever = () => new Promise<never>(vi.fn())

const resolved = {
  id: '01JQVAULTPHYSICS0000000000',
  name: 'Vault',
  path: '/vaults/Physics',
  scan: { isReady: true, error: '', unwatchedPath: '' },
  coverage: '',
}

/** What the core says about the open vault and the window, in the values the window reads them as. */
interface Streams {
  state(): Promise<typeof resolved>
  watchVaultChanges(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    shouldReload: boolean
    renamed: { from: string; to: string }[]
    versionPaths?: string[]
  }>
  watchFocus(signal: AbortSignal): AsyncIterable<{
    path: string
    ranges: { start: number; end: number }[]
    tabId: string
  }>
  watchTasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
}

/** What the core says about the notes while the window follows the vault. */
interface NoteStreams {
  getInitialOpenPath(): Promise<{ path: string } | null>
  watchEdits(): AsyncGenerator<NoteEdit>
}

type Fake = Streams & NoteStreams

let restore: () => void = () => undefined

afterEach(() => {
  restore()
})

const answerFrom = (core: Fake) => {
  restore()
  restore = installCoreAnswers({
    vault: {
      getVaultState: async () => {
        const state = await core.state()
        return {
          id: state.id,
          name: state.name,
          path: state.path,
          scan: {
            isReady: state.scan.isReady,
            error: state.scan.error,
            watchError: state.scan.unwatchedPath,
          },
          coverage: { text: state.coverage },
        }
      },
      watchVaultChanges: async function* (_request, context) {
        for await (const change of core.watchVaultChanges(context.signal)) {
          yield {
            paths: change.paths,
            reload: change.shouldReload,
            renamed: change.renamed,
            versionPaths: change.versionPaths ?? [],
          }
        }
      },
    },
    notes: {
      getOpeningNote: async () => {
        const note = await core.getInitialOpenPath()
        return note ? { note: { path: note.path, title: '' } } : {}
      },
      watchEdits: async function* () {
        for await (const edit of core.watchEdits()) {
          yield {
            change: edit.change,
            path: edit.path,
            range: edit.span,
            text: edit.text,
            isFinal: edit.isComplete,
          }
        }
      },
    },
    workspace: { watchFocus: (_request, context) => core.watchFocus(context.signal) },
    windowService: {
      watchTasks: async function* (_request, context) {
        for await (const list of core.watchTasks(context.signal)) {
          yield {
            tasks: list.map((one) => ({
              id: one.id,
              title: one.label,
              done: BigInt(one.progress.completedCount),
              total: BigInt(one.progress.total),
              unit: TASK_UNIT.THINGS,
              kind: one.kind === 'notice' ? TASK_KIND.NOTICE : TASK_KIND.UNSPECIFIED,
            })),
          }
        }
      },
    },
    pluginHealth: {
      watchHealth: async function* () {
        yield { items: [] }
        await waitForever()
      },
    },
  })
}

const useDisplay = (core: Fake, how?: Parameters<typeof useWindowDisplay>[0]) => {
  answerFrom(core)
  return useWindowDisplay(how)
}

function fake(overrides: Partial<Fake> = {}): Fake {
  return {
    getInitialOpenPath: async () => ({ path: 'Opening.md' }),
    state: async () => resolved,
    watchVaultChanges: emptyStream,
    watchFocus: emptyStream,
    watchEdits: emptyStream,
    watchTasks: async function* () {
      await waitForever()
    },
    ...overrides,
  }
}

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const createDisplay = (
  core: Fake,
  openFileAt?: (path: string, spans: readonly TextRange[]) => void,
) => {
  const changed: string[] = []
  const travelled: string[] = []
  const showed = useDisplay(core, {
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
    const window = useDisplay(core, {
      wait: async (ms) => {
        waits.push(ms)
        if (streams >= 3) window.close()
      },
    })

    await window.streams.follow()

    expect(streams).toBeGreaterThanOrEqual(3)
    expect(waits.length).toBeGreaterThanOrEqual(2)
  })

  it('is taken up again when it fails, and says what happened', async () => {
    let streams = 0
    const core = fake({
      watchVaultChanges: function* () {
        streams++
        throw new Error('connection lost')
      } as unknown as Streams['watchVaultChanges'],
    })
    const window = useDisplay(core, {
      wait: async () => {
        if (streams >= 2) window.close()
      },
    })

    await window.streams.follow()

    expect(streams).toBeGreaterThanOrEqual(2)
    expect(window.health.lost.value).toContain('lost touch with numen')
  })

  it('says what changed, and waits for whatever is drawn from it', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: [], shouldReload: false, renamed: [] }
        yield { paths: ['Somewhere/Else.md'], shouldReload: false, renamed: [] }
      },
    })
    const one = createDisplay(core)

    await one.window.streams.follow()

    expect(one.changed).toStrictEqual(['', 'Somewhere/Else.md'])
  })

  it('says a reload is empty at all, so everything reads again', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: [], shouldReload: false, renamed: [] }
        yield { paths: ['Note.md'], shouldReload: true, renamed: [] }
      },
    })
    const one = createDisplay(core)

    await one.window.streams.follow()

    expect(one.changed).toStrictEqual(['', ''])
  })

  it('says where a note went, for whatever is showing it to follow', async () => {
    const core = fake({
      watchVaultChanges: async function* () {
        yield { paths: [], shouldReload: false, renamed: [] }
        yield { paths: [], shouldReload: false, renamed: [{ from: 'Note.md', to: 'Renamed.md' }] }
      },
    })
    const one = createDisplay(core)

    await one.window.streams.follow()

    expect(one.changed).toStrictEqual(['', 'Note.md → Renamed.md'])
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
    await one.window.opening.read()

    await one.window.streams.follow()

    expect(one.window.opening.path.value).toBe('Renamed.md')
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
    await one.window.opening.read()

    await one.window.streams.follow()

    expect(one.window.opening.path.value).toBe('Opening.md')
    expect(request).toBe(1)
  })
})

/**
 * A vault is swapped from outside this window; the resulting reload is
 * indistinguishable from the one signalling that the whole vault changed on disk.
 */
describe('another vault under this window', () => {
  /** What the window did about a reload: drew the page again, or read again. */
  const createWindow = (core: Fake) => {
    const renderedElement: string[] = []
    const window = useDisplay(core, {
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
  const createReloadingCore = (state: Streams['state'], ahead: readonly string[] = []) =>
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
   * The vault is re-read for every change it reports, and a change of vault arrives at the
   * window before the reload does. The folder the page was rendered on is what the reload
   * is compared against.
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
    expect(one.renderEvents).toContain('changed ')
    expect(one.window.health.lost.value).toBe('')

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
        yield { path: 'Wanted.md', ranges: [], tabId: '' }
      },
    })
    const one = createDisplay(core)

    await one.window.streams.watch()

    expect(one.travelled).toStrictEqual(['Wanted.md'])
  })
})

describe('a tab requested from outside the window', () => {
  it('activates the tab with the id and does not navigate', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield { path: '', ranges: [], tabId: 'tab-2' }
      },
    })
    const activated: string[] = []
    const travelled: string[] = []
    const window = useDisplay(core, {
      wait: async () => window.close(),
      travelTo: (path) => {
        travelled.push(path)
      },
      activateTab: (id) => {
        activated.push(id)
      },
    })

    await window.streams.watch()

    expect(activated).toStrictEqual(['tab-2'])
    expect(travelled).toStrictEqual([])
  })
})

describe('a place inside a source requested for from outside the window', () => {
  /** A window that records the documents it opens, and where. */
  const createRecordingWindow = (core: Fake) => {
    const openedWindows: string[] = []
    const one = createDisplay(core, (path, spans) =>
      openedWindows.push(`${path} ${spans.map((one) => `${one.start} ${one.end}`).join(' ')}`),
    )
    return { window: one.window, opened: openedWindows, travelled: one.travelled }
  }

  it('opens the document it remains in, and asks for no note at all', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield {
          path: 'library/mahabharata.epub',
          ranges: [{ start: 40_512, end: 40_543 }],
          tabId: '',
        }
      },
    })
    const { window, opened, travelled } = createRecordingWindow(core)

    await window.streams.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 40543'])
    expect(travelled).toStrictEqual([])
  })

  it('opens the document at every place the focus names, the first of them first', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield {
          path: 'library/mahabharata.epub',
          ranges: [
            { start: 40_512, end: 40_543 },
            { start: 41_000, end: 41_020 },
            // A span reaching nowhere is no place, and is not lit.
            { start: 42_000, end: 42_000 },
          ],
          tabId: '',
        }
      },
    })
    const { window, opened } = createRecordingWindow(core)

    await window.streams.watch()

    expect(opened).toStrictEqual(['library/mahabharata.epub 40512 40543 41000 41020'])
  })

  it('asks for the note itself where the focus names no run of a source', async () => {
    const core = fake({
      watchFocus: async function* () {
        yield { path: 'Wanted.md', ranges: [{ start: 0, end: 0 }], tabId: '' }
      },
    })
    const { window, opened, travelled } = createRecordingWindow(core)

    await window.streams.watch()

    expect(opened).toStrictEqual([])
    expect(travelled).toStrictEqual(['Wanted.md'])
  })
})

describe('a vault that could not be read', () => {
  it('stops the waiting and is not called empty', async () => {
    const travelled: string[] = []
    const core = fake({
      getInitialOpenPath: async () => null,
      watchVaultChanges: async function* () {
        await waitForever()
      },
      watchEdits: async function* () {
        await waitForever()
      },
      watchFocus: async function* () {
        yield { path: 'Focused.md', ranges: [], tabId: '' }
      },
      state: async () => ({
        ...resolved,
        scan: { isReady: false, error: 'permission denied', unwatchedPath: '' },
      }),
    })
    const window = useDisplay(core, {
      wait: async () => window.close(),
      travelTo: (path) => {
        travelled.push(path)
      },
    })

    await window.start()
    await flushPromises()

    expect(window.health.isIndexing.value).toBe(false)
    expect(window.vault.error.value).toBe('permission denied')
    expect(window.opening.hasNote.value).toBe(false)
    await vi.waitFor(() => expect(travelled).toStrictEqual(['Focused.md']))
  })
})

describe('a vault with a note in it', () => {
  it('says it holds one, which is not what an unreadable vault says', async () => {
    const window = useDisplay(fake(), { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.opening.hasNote.value).toBe(true)
    expect(window.vault.error.value).toBe('')
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
    const window = useDisplay(core, { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.vault.unwatched.value).toBe('too many watches')
  })
})

describe('what the search source says of its index', () => {
  it('is kept as it is given', async () => {
    const core = fake({
      state: async () => ({ ...resolved, coverage: 'Semantic index not built yet' }),
    })
    const window = useDisplay(core, { wait: async () => window.close() })

    await window.start()
    await flushPromises()

    expect(window.vault.coverage.value).toBe('Semantic index not built yet')
  })
})

describe('what the application is doing', () => {
  const readText = (count: number): Task => ({
    id: 'reading:library/scan.pdf',
    label: 'Recognising a scan',
    about: 'library/scan.pdf',
    progress: { completedCount: count, total: 400, counting: 'items' },
    error: '',
    source: { isUserStarted: true, pluginId: '', pluginName: '' },
    isCancellable: false,
    kind: 'work',
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
    const window = useDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()

    await vi.waitFor(() =>
      expect(window.tasks.value.map((one) => one.progress.completedCount)).toEqual([32]),
    )

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
    const window = useDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()
    await flushPromises()

    await vi.waitFor(() =>
      expect(window.tasks.value.map((one) => one.progress.completedCount)).toEqual([48]),
    )
    expect(openedWindows).toBeGreaterThan(1)
    // A stream taken up again is not a stream that was lost.
    expect(window.health.lost.value).toBe('')

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
          coverage: `Indexed ${asks * 1000} passages`,
        }
      },
      watchTasks: async function* () {
        yield [readText(16)]
        yield []
        await waitForever()
      },
    })
    const window = useDisplay(core, { wait: vi.fn(() => Promise.resolve()) })

    await window.start()
    await flushPromises()

    expect(asks).toBeGreaterThan(1)
    expect(window.vault.coverage.value).toBe(`Indexed ${asks * 1000} passages`)

    window.close()
  })
})

describe('a window before it has requested the vault anything', () => {
  it('says nothing yet, and is waiting for the vault to be read', () => {
    const window = useDisplay(fake())

    expect(window.vault.name.value).toBe('')
    expect(window.health.isIndexing.value).toBe(true)
    expect(window.health.failure.value).toBe('')
    expect(window.health.lost.value).toBe('')
    expect(window.vault.error.value).toBe('')
    expect(window.vault.unwatched.value).toBe('')
    expect(window.opening.hasNote.value).toBe(false)
    expect(window.opening.path.value).toBe('')
    expect(window.vault.coverage.value).toBe('')
    expect(window.tasks.value).toStrictEqual([])
  })
})

describe('the note the vault opens with', () => {
  it('is none for a vault that holds no note', async () => {
    const window = useDisplay(fake({ getInitialOpenPath: async () => null }))

    expect(await window.opening.read()).toBe('')
    expect(window.opening.path.value).toBe('')
    expect(window.opening.hasNote.value).toBe(false)
  })
})

describe('the first note shown', () => {
  it('stops the waiting and notifies everything showing the vault to read it whole', async () => {
    const one = createDisplay(fake())

    await one.window.start()
    await flushPromises()

    expect(one.window.health.isIndexing.value).toBe(false)
    expect(one.changed).toStrictEqual([''])
  })

  it('says what changed in a note while it is being made', async () => {
    const edit = {
      change: 'c1',
      path: 'Note.md',
      span: { start: 0, end: 4 },
      text: 'Leaf',
      isComplete: false,
    }
    const response: unknown[] = []
    const core = fake({
      watchEdits: async function* () {
        yield edit
      },
    })
    const window = useDisplay(core, {
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
    const window = useDisplay(core, {
      onVaultChanged: (paths) => {
        changed.push(paths)
      },
    })

    await window.start()
    await flushPromises()

    expect(window.health.isIndexing.value).toBe(true)
    expect(changed).toStrictEqual([])
  })

  it('is waited for while the vault holds none and is still being read', async () => {
    const changed: unknown[] = []
    const core = fake({
      getInitialOpenPath: async () => null,
      state: async () => ({ ...resolved, scan: { ...resolved.scan, isReady: false } }),
    })
    const waits: number[] = []
    const window = useDisplay(core, {
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

    expect(window.health.isIndexing.value).toBe(true)
    expect(changed).toStrictEqual([])
    expect(waits).toStrictEqual([100])
  })
})

describe('a closed window', () => {
  it('releases every stream it listens to', async () => {
    const signals: AbortSignal[] = []
    const core = fake({
      watchTasks: async function* (signal) {
        signals.push(signal)
        await waitForever()
      },
    })
    const window = useDisplay(core, { wait: waitForever })

    await window.start()
    await flushPromises()
    window.close()

    expect(signals).toHaveLength(1)
    expect(signals[0]?.aborted).toBe(true)
  })
})

describe('a vault that cannot be reached at all', () => {
  it('says so, and stops the waiting', async () => {
    const window = useDisplay(
      fake({
        state: async () => {
          throw new Error('connection failed')
        },
      }),
    )

    await window.start()

    expect(window.health.failure.value).not.toBe('')
    expect(window.health.isIndexing.value).toBe(false)
  })
})

describe('a vault still being read', () => {
  it('is requested again a tenth of a second later', async () => {
    vi.useFakeTimers()
    let reads = 0
    const window = useDisplay(
      fake({
        getInitialOpenPath: async () => null,
        state: async () => {
          reads += 1
          return { ...resolved, scan: { ...resolved.scan, isReady: false } }
        },
      }),
    )

    const started = window.start()
    await vi.advanceTimersByTimeAsync(0)
    expect(reads).toBe(1)
    await vi.advanceTimersByTimeAsync(100)
    window.close()
    await vi.advanceTimersByTimeAsync(100)
    await started
    vi.useRealTimers()

    expect(reads).toBe(2)
  })
})
