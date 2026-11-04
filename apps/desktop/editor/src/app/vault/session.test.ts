/**
 * The vault state response mapped to window values: a field omitted from the
 * response maps to an empty or zero value.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FLUSH_RESULT, TASK_UNIT } from '@numen/core-client'
import { sessionCore } from './session'

const request = vi.hoisted(() => ({
  getVaultState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
  watchVaultChanges: vi.fn(),
  watchFocus: vi.fn(),
  watchTasks: vi.fn(),
  cancelTask: vi.fn(),
  watchQuit: vi.fn(),
  detachTab: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({
  vault: request,
  agentService: request,
  workspace: request,
  windowService: request,
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe('the state of the vault', () => {
  it('is a scan not begun and nothing indexed where the answer carries neither', async () => {
    request.getVaultState.mockResolvedValue({ id: 'v1', name: 'Zoo', path: '/zoo' })

    expect(await sessionCore.state()).toStrictEqual({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: false, error: '', unwatchedPath: '' },
      coverage: { chunkCount: 0n, embeddedCount: 0n, isEmbedding: false },
    })
  })

  it('carries how far the scan got and how much the index holds', async () => {
    request.getVaultState.mockResolvedValue({
      id: 'v1',
      name: 'Zoo',
      path: '/zoo',
      scan: { isReady: true, error: 'permission denied', watchError: '/zoo/deep' },
      coverage: { chunkCount: 12n, embeddedCount: 5n, isEmbedding: true },
    })

    const state = await sessionCore.state()

    expect(state.scan).toStrictEqual({
      isReady: true,
      error: 'permission denied',
      unwatchedPath: '/zoo/deep',
    })
    expect(state.coverage).toStrictEqual({ chunkCount: 12n, embeddedCount: 5n, isEmbedding: true })
  })
})

describe('the tabs a window holds open', () => {
  it('are written as they stand, with the one in front', async () => {
    const tab = { id: 't1', kind: 'note', path: 'Llama.md' }

    await sessionCore.writeOpenTabs({ tabs: [tab], activeId: 't1' } as never)

    expect(request.writeOpenTabs).toHaveBeenCalledWith({ tabs: [tab], activeId: 't1' })
  })
})

describe('a flush reported', () => {
  it('says nothing was left where the window had nothing unsaved', async () => {
    await sessionCore.reportFlush('token-1')

    expect(request.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FLUSH_RESULT.NOTHING,
    })
  })

  it('says what the window wrote', async () => {
    await sessionCore.reportFlush('token-1', 'written')

    expect(request.reportFlush).toHaveBeenCalledWith({
      window: 'editor',
      token: 'token-1',
      result: FLUSH_RESULT.WRITTEN,
    })
  })
})

/** Everything a stream gave, read to its end. */
const readAll = async <T>(stream: AsyncIterable<T>): Promise<T[]> => {
  const readItem: T[] = []
  for await (const one of stream) readItem.push(one)
  return readItem
}

describe('the streams a window follows', () => {
  const signal = new AbortController().signal

  it('notifies each change of the vault, and where every note named went', async () => {
    request.watchVaultChanges.mockImplementation(async function* () {
      yield { paths: ['Llama.md'], reload: true, renamed: [{ from: 'A.md', to: 'B.md' }] }
    })

    expect(await readAll(sessionCore.watchVaultChanges(signal))).toStrictEqual([
      { paths: ['Llama.md'], shouldReload: true, renamed: [{ from: 'A.md', to: 'B.md' }] },
    ])
    expect(request.watchVaultChanges).toHaveBeenCalledWith({}, { signal })
  })

  it('follows the focus, and lets go of it on the signal', () => {
    const focus = (async function* () {
      yield* []
    })()
    request.watchFocus.mockReturnValue(focus)

    expect(sessionCore.watchFocus(signal)).toBe(focus)
    expect(request.watchFocus).toHaveBeenCalledWith({}, { signal })
  })

  it('follows a quit requested of this window', () => {
    const quit = (async function* () {
      yield* []
    })()
    request.watchQuit.mockReturnValue(quit)

    expect(sessionCore.watchQuit(signal)).toBe(quit)
    expect(request.watchQuit).toHaveBeenCalledWith({ window: 'editor' }, { signal })
  })

  it('notifies the work under way, counted in what each counts', async () => {
    request.watchTasks.mockImplementation(async function* () {
      yield {
        tasks: [
          {
            id: 'scan',
            title: 'Recognising a scan',
            about: 'library/scan.pdf',
            done: 3n,
            total: 10n,
            unit: TASK_UNIT.BYTES,
            error: '',
            isUserStarted: true,
            pluginId: 'md.example.scans',
            pluginName: 'Scan reader',
            isCancellable: true,
          },
          {
            id: 'later',
            title: 'Later',
            about: '',
            done: 0n,
            total: 0n,
            unit: 99,
            error: 'stopped',
            isUserStarted: false,
            pluginId: '',
            pluginName: '',
            isCancellable: false,
          },
        ],
      }
    })

    expect(await readAll(sessionCore.watchTasks(signal))).toStrictEqual([
      [
        {
          id: 'scan',
          label: 'Recognising a scan',
          about: 'library/scan.pdf',
          completedCount: 3,
          total: 10,
          counting: 'bytes',
          error: '',
          isUserStarted: true,
          pluginId: 'md.example.scans',
          pluginName: 'Scan reader',
          isCancellable: true,
        },
        {
          id: 'later',
          label: 'Later',
          about: '',
          completedCount: 0,
          total: 0,
          counting: 'things',
          error: 'stopped',
          isUserStarted: false,
          pluginId: '',
          pluginName: '',
          isCancellable: false,
        },
      ],
    ])
    expect(request.watchTasks).toHaveBeenCalledWith({ window: 'editor' }, { signal })
  })

  it('asks the core of this window to cancel a task by its id', async () => {
    request.cancelTask.mockResolvedValue({})

    await sessionCore.cancelTask('making the vectors')

    expect(request.cancelTask).toHaveBeenCalledWith({ window: 'editor', id: 'making the vectors' })
  })

  it('rejects when the core does not cancel the task', async () => {
    request.cancelTask.mockRejectedValue(new Error('not found'))

    await expect(sessionCore.cancelTask('gone')).rejects.toThrow('not found')
  })
})

describe('detaching a tab', () => {
  it('asks the workspace to open a new window for the tab', async () => {
    request.detachTab.mockResolvedValue({})
    await sessionCore.detachTab(
      { id: 'plex:3b9d', kind: 'plex', path: 'Physics.md' },
      { x: 100.6, y: 200.4 },
    )

    expect(request.detachTab).toHaveBeenCalledWith({
      id: 'plex:3b9d',
      kind: 'plex',
      path: 'Physics.md',
      x: 101,
      y: 200,
    })
  })
})
