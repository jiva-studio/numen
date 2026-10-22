import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { useWindowStreams, type WindowStreamsDeps } from './useWindowStreams'
import type { Task } from '@/shared/notices/task'

/** A stream that yields the given items and then ends. */
async function* yielding<T>(...items: T[]): AsyncIterable<T> {
  for (const item of items) yield item
}

/** Deps whose window closes after the first reading of a stream ends. */
const setup = (overrides: Partial<WindowStreamsDeps> = {}) => {
  let open = true
  const deps = {
    core: {
      watchVaultChanges: () => yielding(),
      watchFocus: () => yielding(),
      watchTasks: () => yielding(),
      watchEdits: () => yielding(),
    },
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
    onVaultChanged: vi.fn(async () => {}),
    readInitialNote: vi.fn(async () => ''),
    ask: vi.fn(async () => {}),
    reportChange: vi.fn(),
    travelTo: vi.fn(),
    openFileAt: vi.fn(),
    ...overrides,
  } as WindowStreamsDeps
  return deps
}

const task = (id: string): Task => ({
  id,
  label: 'Indexing',
  about: '',
  completedCount: 0,
  total: 0,
  counting: 'items',
  error: '',
  isUserStarted: false,
})

describe('following the vault', () => {
  it('ignores a change that names no path, no reload and no rename', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () => yielding({ paths: [], shouldReload: false, renamed: [] }),
      },
    })

    await useWindowStreams(deps).follow()

    expect(deps.onVaultChanged).not.toHaveBeenCalled()
    expect(deps.ask).not.toHaveBeenCalled()
  })

  it.each([
    ['a path', { paths: ['a.md'], shouldReload: false, renamed: [] }],
    ['a reload', { paths: [], shouldReload: true, renamed: [] }],
    ['a rename', { paths: [], shouldReload: false, renamed: [{ from: 'a.md', to: 'b.md' }] }],
  ])('tells a change that names %s', async (_name, change) => {
    const deps = setup({
      core: { ...setup().core, watchVaultChanges: () => yielding(change) },
    })

    await useWindowStreams(deps).follow()

    expect(deps.onVaultChanged).toHaveBeenCalledTimes(1)
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('asks for the counts again after a change, and an error does not end the stream', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () => yielding({ paths: ['a.md'], shouldReload: false, renamed: [] }),
      },
      ask: vi.fn(async () => {
        throw new Error('failed')
      }),
    })

    await useWindowStreams(deps).follow()

    expect(deps.ask).toHaveBeenCalledTimes(1)
    expect(deps.setLost).not.toHaveBeenCalledWith(expect.stringContaining('lost'))
  })

  it('reads the opening note again when that very note is renamed, and not another', async () => {
    const renamedOpening = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () =>
          yielding({
            paths: [],
            shouldReload: false,
            renamed: [
              { from: 'Other.md', to: 'Moved.md' },
              { from: 'Opening.md', to: 'Renamed.md' },
            ],
          }),
      },
    })
    await useWindowStreams(renamedOpening).follow()
    expect(renamedOpening.readInitialNote).toHaveBeenCalledTimes(1)

    const renamedOther = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () =>
          yielding({
            paths: [],
            shouldReload: false,
            renamed: [
              { from: 'Other.md', to: 'Moved.md' },
              { from: 'Opening.md', to: 'Renamed.md' },
              { from: 'Third.md', to: 'Fourth.md' },
            ],
          }),
      },
    })
    renamedOther.opening.value = 'Elsewhere.md'
    await useWindowStreams(renamedOther).follow()
    expect(renamedOther.readInitialNote).not.toHaveBeenCalled()
  })

  it('survives an error when the opening note is read again', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () =>
          yielding({
            paths: [],
            shouldReload: false,
            renamed: [
              { from: 'Other.md', to: 'Moved.md' },
              { from: 'Opening.md', to: 'Renamed.md' },
            ],
          }),
      },
      readInitialNote: vi.fn(async () => {
        throw new Error('gone')
      }),
    })

    await useWindowStreams(deps).follow()

    expect(deps.readInitialNote).toHaveBeenCalledTimes(1)
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('draws the page again on a reload that swaps the vault, and notifies nothing', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchVaultChanges: () => yielding({ paths: [], shouldReload: true, renamed: [] }),
      },
      isVaultSwapped: vi.fn(async () => true),
    })

    await useWindowStreams(deps).follow()

    expect(deps.reload).toHaveBeenCalledTimes(1)
    expect(deps.onVaultChanged).not.toHaveBeenCalled()
  })
})

describe('following the edits', () => {
  it('reports an edit that names a path, and not the opening silence', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchEdits: () => yielding({ path: '' }, { path: 'a.md' }),
      } as unknown as WindowStreamsDeps['core'],
    })

    await useWindowStreams(deps).draw()

    expect(deps.reportChange).toHaveBeenCalledTimes(1)
    expect(deps.reportChange).toHaveBeenCalledWith({ path: 'a.md' })
  })
})

describe('following the focus', () => {
  it('travels nowhere on a focus that names no path', async () => {
    const deps = setup({
      core: { ...setup().core, watchFocus: () => yielding({ path: '', spans: [] }) },
    })

    await useWindowStreams(deps).watch()

    expect(deps.travelTo).not.toHaveBeenCalled()
    expect(deps.openFileAt).not.toHaveBeenCalled()
  })

  it('travels to the path of a focus that names no span, and opens one that names a span', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchFocus: () =>
          yielding(
            { path: 'a.md', spans: [] },
            { path: 'b.md', spans: [{ from: 3, to: 3 }] },
            { path: 'c.md', spans: [{ from: 2, to: 9 }] },
          ),
      },
    })

    await useWindowStreams(deps).watch()

    expect(deps.travelTo).toHaveBeenCalledTimes(2)
    expect(deps.travelTo).toHaveBeenNthCalledWith(1, 'a.md')
    expect(deps.travelTo).toHaveBeenNthCalledWith(2, 'b.md')
    expect(deps.openFileAt).toHaveBeenCalledTimes(1)
    expect(deps.openFileAt).toHaveBeenCalledWith('c.md', [{ from: 2, to: 9 }])
  })
})

describe('following the tasks', () => {
  it('keeps the list it is given and asks nothing of the first one', async () => {
    const deps = setup({
      core: { ...setup().core, watchTasks: () => yielding([]) },
    })

    await useWindowStreams(deps).followTasks()

    expect(deps.ask).not.toHaveBeenCalled()
  })

  it('asks for the counts again when a list that retained work empties, and only then', async () => {
    const deps = setup({
      core: {
        ...setup().core,
        watchTasks: () => yielding([task('a')], [task('a'), task('b')], [], []),
      },
    })

    await useWindowStreams(deps).followTasks()

    expect(deps.tasks.value).toEqual([])
    expect(deps.ask).toHaveBeenCalledTimes(1)
  })

  it('does not ask while the list still holds work', async () => {
    const deps = setup({
      core: { ...setup().core, watchTasks: () => yielding([task('a')], [task('b')]) },
    })

    await useWindowStreams(deps).followTasks()

    expect(deps.tasks.value).toEqual([task('b')])
    expect(deps.ask).not.toHaveBeenCalled()
  })

  it('survives an error of the counts', async () => {
    const deps = setup({
      core: { ...setup().core, watchTasks: () => yielding([task('a')], []) },
      ask: vi.fn(async () => {
        throw new Error('failed')
      }),
    })

    await useWindowStreams(deps).followTasks()

    expect(deps.ask).toHaveBeenCalledTimes(1)
  })
})
