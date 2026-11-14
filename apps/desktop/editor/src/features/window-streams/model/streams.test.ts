import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import { TASK_UNIT } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import { createFollower } from '@numen/ui'
import { createEditStream, createFocusStream, createTaskStream, createVaultStream } from './streams'
import type { NoteEdit } from '@/entities/note'
import type { Task } from '@/entities/window'
import type { TextRange } from '@/shared/lib/textRange'

async function* yielding<T>(...items: T[]): AsyncIterable<T> {
  for (const item of items) yield item
}

interface Change {
  paths: string[]
  shouldReload: boolean
  renamed: { from: string; to: string }[]
  versionPaths?: string[]
}

interface Focus {
  path: string
  ranges: { start: number; end: number }[]
  tabId: string
}

const task = (id: string): Task => ({
  id,
  label: 'Indexing',
  about: '',
  progress: { completedCount: 0, total: 0, counting: 'items' },
  error: '',
  source: { isUserStarted: false, pluginId: '', pluginName: '' },
  isCancellable: false,
  kind: 'work',
})

/** What the core sends for the streams the window follows. A stream not named sends nothing. */
interface Streams {
  changes?: Change[]
  focus?: Focus[]
  tasks?: Task[][]
  edits?: { change: string; path: string; range?: { start: number; end: number }; text: string }[]
}

let restore: () => void = () => undefined

afterEach(() => {
  restore()
})

const answer = (streams: Streams) => {
  restore = installCoreAnswers({
    vault: {
      watchVaultChanges: () =>
        yielding(
          ...(streams.changes ?? []).map((one) => ({
            paths: one.paths,
            reload: one.shouldReload,
            renamed: one.renamed,
            versionPaths: one.versionPaths ?? [],
          })),
        ),
    },
    workspace: { watchFocus: () => yielding(...(streams.focus ?? [])) },
    notes: {
      watchEdits: () =>
        yielding(...(streams.edits ?? []).map((one) => ({ ...one, isFinal: false }))),
    },
    windowService: {
      watchTasks: () =>
        yielding(
          ...(streams.tasks ?? []).map((list) => ({
            tasks: list.map((one) => ({
              id: one.id,
              title: one.label,
              unit: TASK_UNIT.THINGS,
              error: one.error,
            })),
          })),
        ),
    },
  })
}

/** Everything the streams of the window are made over, as the tests set it. */
interface WindowStreamsDeps {
  readonly listening: AbortController
  readonly isOpen: () => boolean
  readonly setLost: (response: string) => void
  readonly wait: (ms: number) => Promise<void>
  readonly opening: Ref<string>
  readonly tasks: ShallowRef<readonly Task[]>
  readonly isVaultSwapped: () => Promise<boolean>
  readonly reload: () => void
  readonly onVaultChanged: (
    paths: readonly string[],
    renamed: readonly { from: string; to: string }[],
    versionPaths: readonly string[],
  ) => Promise<void> | void
  readonly readInitialNote: () => Promise<string>
  readonly ask: () => Promise<{ scan: { error?: string; isReady?: boolean } }>
  readonly reportChange: (edit: NoteEdit) => void
  readonly travelTo: (path: string) => Promise<void> | void
  readonly openFileAt: (path: string, spans: readonly TextRange[]) => void
  readonly activateTab: (id: string) => void
}

const createWindowStreams = (deps: WindowStreamsDeps) => {
  const follows = createFollower({ isOpen: deps.isOpen, setLost: deps.setLost, wait: deps.wait })
  return {
    follow: createVaultStream({ ...deps, follows }),
    draw: createEditStream({ ...deps, follows }),
    watch: createFocusStream({ ...deps, follows }),
    followTasks: createTaskStream({ ...deps, follows }),
  }
}

const setup = (streams: Streams = {}, overrides: Partial<WindowStreamsDeps> = {}) => {
  answer(streams)
  let open = true
  const deps = {
    listening: new AbortController(),
    isOpen: () => open,
    setLost: vi.fn(),
    wait: vi.fn(async () => {
      open = false
    }),
    opening: ref('Opening.md'),
    tasks: shallowRef<readonly Task[]>([]),
    isVaultSwapped: vi.fn(async () => false),
    reload: vi.fn(),
    onVaultChanged: vi.fn(() => Promise.resolve()),
    readInitialNote: vi.fn(async () => ''),
    ask: vi.fn(() => Promise.resolve()),
    reportChange: vi.fn(),
    travelTo: vi.fn(),
    openFileAt: vi.fn(),
    activateTab: vi.fn(),
    ...overrides,
  } as WindowStreamsDeps
  return deps
}

const opened: Change = { paths: [], shouldReload: false, renamed: [] }

describe('following the vault', () => {
  it('passes on a change that names only files whose text versions changed', async () => {
    const deps = setup({
      changes: [
        opened,
        { paths: [], shouldReload: false, renamed: [], versionPaths: ['talks/a.mp3'] },
      ],
    })

    await createWindowStreams(deps).follow()

    expect(deps.onVaultChanged).toHaveBeenLastCalledWith([], [], ['talks/a.mp3'])
  })

  it('lists the files again when the stream opens, and asks nothing more', async () => {
    const deps = setup({ changes: [opened] })

    await createWindowStreams(deps).follow()

    expect(deps.onVaultChanged).toHaveBeenCalledExactlyOnceWith([], [], [])
    expect(deps.ask).not.toHaveBeenCalled()
  })

  it('ignores a change that names no path, no reload and no rename after the stream opened', async () => {
    const deps = setup({ changes: [opened, opened] })

    await createWindowStreams(deps).follow()

    expect(deps.onVaultChanged).toHaveBeenCalledTimes(1)
    expect(deps.ask).not.toHaveBeenCalled()
  })

  it.each([
    ['a path', { paths: ['a.md'], shouldReload: false, renamed: [] }],
    ['a reload', { paths: [], shouldReload: true, renamed: [] }],
    ['a rename', { paths: [], shouldReload: false, renamed: [{ from: 'a.md', to: 'b.md' }] }],
  ])('tells a change that names %s', async (_name, change) => {
    const deps = setup({ changes: [opened, change] })

    await createWindowStreams(deps).follow()

    expect(deps.onVaultChanged).toHaveBeenCalledTimes(2)
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('asks for the counts again after a change, and an error does not end the stream', async () => {
    const deps = setup(
      { changes: [{ paths: ['a.md'], shouldReload: false, renamed: [] }] },
      {
        ask: vi.fn(async () => {
          throw new Error('failed')
        }),
      },
    )

    await createWindowStreams(deps).follow()

    expect(deps.ask).toHaveBeenCalledTimes(1)
    expect(deps.setLost).not.toHaveBeenCalledWith(expect.stringContaining('lost'))
  })

  it('reads the opening note again when that very note is renamed, and not another', async () => {
    const renamedOpening = setup({
      changes: [
        {
          paths: [],
          shouldReload: false,
          renamed: [
            { from: 'Other.md', to: 'Moved.md' },
            { from: 'Opening.md', to: 'Renamed.md' },
          ],
        },
      ],
    })
    await createWindowStreams(renamedOpening).follow()
    expect(renamedOpening.readInitialNote).toHaveBeenCalledTimes(1)

    const renamedOther = setup({
      changes: [
        {
          paths: [],
          shouldReload: false,
          renamed: [
            { from: 'Other.md', to: 'Moved.md' },
            { from: 'Opening.md', to: 'Renamed.md' },
            { from: 'Third.md', to: 'Fourth.md' },
          ],
        },
      ],
    })
    renamedOther.opening.value = 'Elsewhere.md'
    await createWindowStreams(renamedOther).follow()
    expect(renamedOther.readInitialNote).not.toHaveBeenCalled()
  })

  it('survives an error when the opening note is read again', async () => {
    const deps = setup(
      {
        changes: [
          {
            paths: [],
            shouldReload: false,
            renamed: [
              { from: 'Other.md', to: 'Moved.md' },
              { from: 'Opening.md', to: 'Renamed.md' },
            ],
          },
        ],
      },
      {
        readInitialNote: vi.fn(async () => {
          throw new Error('gone')
        }),
      },
    )

    await createWindowStreams(deps).follow()

    expect(deps.readInitialNote).toHaveBeenCalledTimes(1)
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('draws the page again on a reload that swaps the vault, and notifies no change', async () => {
    const deps = setup(
      { changes: [opened, { paths: [], shouldReload: true, renamed: [] }] },
      { isVaultSwapped: vi.fn(async () => true) },
    )

    await createWindowStreams(deps).follow()

    expect(deps.reload).toHaveBeenCalledTimes(1)
    expect(deps.onVaultChanged).toHaveBeenCalledExactlyOnceWith([], [], [])
  })
})

describe('following the edits', () => {
  it('reports an edit that names a path, and not the opening silence', async () => {
    const deps = setup({
      edits: [
        { change: '', path: '', text: '' },
        { change: '1', path: 'a.md', range: { start: 2, end: 5 }, text: 'abc' },
      ],
    })

    await createWindowStreams(deps).draw()

    expect(deps.reportChange).toHaveBeenCalledTimes(1)
    expect(deps.reportChange).toHaveBeenCalledWith({
      change: '1',
      path: 'a.md',
      span: { start: 2, end: 5 },
      text: 'abc',
      isComplete: false,
    })
  })
})

describe('following the focus', () => {
  it('travels nowhere on a focus that names no path', async () => {
    const deps = setup({ focus: [{ path: '', ranges: [], tabId: '' }] })

    await createWindowStreams(deps).watch()

    expect(deps.travelTo).not.toHaveBeenCalled()
    expect(deps.openFileAt).not.toHaveBeenCalled()
  })

  it('travels to the path of a focus that names no span, and opens one that names a span', async () => {
    const deps = setup({
      focus: [
        { path: 'a.md', ranges: [], tabId: '' },
        { path: 'b.md', ranges: [{ start: 3, end: 3 }], tabId: '' },
        { path: 'c.md', ranges: [{ start: 2, end: 9 }], tabId: '' },
      ],
    })

    await createWindowStreams(deps).watch()

    expect(deps.travelTo).toHaveBeenCalledTimes(2)
    expect(deps.travelTo).toHaveBeenNthCalledWith(1, 'a.md')
    expect(deps.travelTo).toHaveBeenNthCalledWith(2, 'b.md')
    expect(deps.openFileAt).toHaveBeenCalledTimes(1)
    expect(deps.openFileAt).toHaveBeenCalledWith('c.md', [{ start: 2, end: 9 }])
  })

  it('activates the tab of a focus with a tab id, and does not navigate or open a file', async () => {
    const deps = setup({ focus: [{ path: '', ranges: [], tabId: 'tab-2' }] })

    await createWindowStreams(deps).watch()

    expect(deps.activateTab).toHaveBeenCalledTimes(1)
    expect(deps.activateTab).toHaveBeenCalledWith('tab-2')
    expect(deps.travelTo).not.toHaveBeenCalled()
    expect(deps.openFileAt).not.toHaveBeenCalled()
  })

  it('does not activate a tab on a focus that names a path', async () => {
    const deps = setup({ focus: [{ path: 'a.md', ranges: [], tabId: '' }] })

    await createWindowStreams(deps).watch()

    expect(deps.activateTab).not.toHaveBeenCalled()
    expect(deps.travelTo).toHaveBeenCalledWith('a.md')
  })
})

describe('following the tasks', () => {
  it('keeps the list it is given and asks nothing of the first one', async () => {
    const deps = setup({ tasks: [[]] })

    await createWindowStreams(deps).followTasks()

    expect(deps.ask).not.toHaveBeenCalled()
  })

  it('asks for the counts again when a list that retained work empties, and only then', async () => {
    const deps = setup({ tasks: [[task('a')], [task('a'), task('b')], [], []] })

    await createWindowStreams(deps).followTasks()

    expect(deps.tasks.value).toEqual([])
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('does not ask while the list still holds work', async () => {
    const deps = setup({ tasks: [[task('a')], [task('b')]] })

    await createWindowStreams(deps).followTasks()

    expect(deps.tasks.value).toEqual([task('b')])
    expect(deps.ask).not.toHaveBeenCalled()
  })

  it('survives an error of the counts', async () => {
    const deps = setup(
      { tasks: [[task('a')], []] },
      {
        ask: vi.fn(async () => {
          throw new Error('failed')
        }),
      },
    )

    await createWindowStreams(deps).followTasks()

    expect(deps.ask).toHaveBeenCalledTimes(1)
  })
})
