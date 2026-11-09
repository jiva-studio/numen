/**
 * The vault state response mapped to window values: a field omitted from the
 * response maps to an empty or zero value.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { FLUSH_RESULT, TASK_KIND, TASK_UNIT } from '@numen/core-client'
import { createLostConnection, installCoreHandlers } from '@numen/core-client/testing'
import { sessionCore } from './session'

const request = {
  getVaultState: vi.fn(),
  writeOpenTabs: vi.fn(),
  reportFlush: vi.fn(),
  watchVaultChanges: vi.fn(),
  watchFocus: vi.fn(),
  watchTasks: vi.fn(),
  cancelTask: vi.fn(),
  watchQuit: vi.fn(),
  detachTab: vi.fn(),
}

let restore: () => void

beforeEach(() => {
  vi.clearAllMocks()
  restore = installCoreHandlers(request)
})

afterEach(() => {
  restore()
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

    expect(request.writeOpenTabs.mock.lastCall?.[0]).toMatchObject({ tabs: [tab], activeId: 't1' })
  })
})

describe('a flush reported', () => {
  it('says nothing was left where the window had nothing unsaved', async () => {
    await sessionCore.reportFlush('token-1')

    expect(request.reportFlush.mock.lastCall?.[0]).toMatchObject({
      window: 'editor',
      token: 'token-1',
      result: FLUSH_RESULT.NOTHING,
    })
  })

  it('says what the window wrote', async () => {
    await sessionCore.reportFlush('token-1', 'written')

    expect(request.reportFlush.mock.lastCall?.[0]).toMatchObject({
      window: 'editor',
      token: 'token-1',
      result: FLUSH_RESULT.WRITTEN,
    })
  })
})

const readAll = async <T>(stream: AsyncIterable<T>): Promise<T[]> => {
  const readItem: T[] = []
  for await (const one of stream) readItem.push(one)
  return readItem
}

describe('the streams a window follows', () => {
  const signal = new AbortController().signal

  it('notifies each change of the vault, where every note named went, and the files whose text versions changed', async () => {
    request.watchVaultChanges.mockImplementation(async function* () {
      yield {
        paths: ['Llama.md'],
        reload: true,
        renamed: [{ from: 'A.md', to: 'B.md' }],
        versionPaths: ['talks/a.mp3'],
      }
    })

    expect(await readAll(sessionCore.watchVaultChanges(signal))).toStrictEqual([
      {
        paths: ['Llama.md'],
        shouldReload: true,
        renamed: [{ from: 'A.md', to: 'B.md' }],
        versionPaths: ['talks/a.mp3'],
      },
    ])
  })

  it('follows the focus', async () => {
    request.watchFocus.mockImplementation(async function* () {
      yield {}
    })

    expect(await readAll(sessionCore.watchFocus(signal))).toHaveLength(1)
  })

  it('releases the focus on the signal', async () => {
    let requestSignal: AbortSignal | undefined
    request.watchFocus.mockImplementation(async function* (_request, context) {
      requestSignal = context.signal
      await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
      yield* [] as object[]
    })
    const controller = new AbortController()

    const readTask = readAll(sessionCore.watchFocus(controller.signal))
    await vi.waitFor(() => expect(requestSignal).toBeDefined())
    controller.abort()

    await vi.waitFor(() => expect(requestSignal?.aborted).toBe(true))
    await readTask.catch(() => undefined)
  })

  it('follows a quit requested of this window', async () => {
    request.watchQuit.mockImplementation(async function* () {
      yield {}
    })

    expect(await readAll(sessionCore.watchQuit(signal))).toHaveLength(1)
    expect(request.watchQuit.mock.lastCall?.[0]).toMatchObject({ window: 'editor' })
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
            kind: TASK_KIND.NOTICE,
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
            kind: TASK_KIND.UNSPECIFIED,
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
          kind: 'notice',
        },
        {
          id: 'later',
          label: 'Later',
          about: '',
          completedCount: 0,
          total: 0,
          counting: 'items',
          error: 'stopped',
          isUserStarted: false,
          pluginId: '',
          pluginName: '',
          isCancellable: false,
          kind: 'work',
        },
      ],
    ])
    expect(request.watchTasks.mock.lastCall?.[0]).toMatchObject({ window: 'editor' })
  })

  it('asks the core of this window to cancel a task by its id', async () => {
    request.cancelTask.mockResolvedValue({})

    await sessionCore.cancelTask('making the vectors')

    expect(request.cancelTask.mock.lastCall?.[0]).toMatchObject({
      window: 'editor',
      id: 'making the vectors',
    })
  })

  it('rejects when the core does not cancel the task', async () => {
    request.cancelTask.mockRejectedValue(createLostConnection('not found'))

    await expect(sessionCore.cancelTask('gone')).rejects.toThrow('not found')
  })
})

describe('detaching a tab', () => {
  it('sends DetachTab with the tab, its saved state and the rounded position and size', async () => {
    request.detachTab.mockResolvedValueOnce({ error: 0 })
    request.detachTab.mockResolvedValueOnce({ error: 2 })
    const tab = {
      id: 'plex:3b9d',
      kind: 'plex',
      path: 'Physics.md',
      title: 'Physics',
      state: '{"focus":"Kepler"}',
    }
    expect(
      await sessionCore.detachTab(tab, { x: 100.6, y: 200.4, width: 640.5, height: 479.6 }),
    ).toStrictEqual({
      ok: true,
      value: undefined,
    })
    expect(await sessionCore.detachTab(tab, { x: 0, y: 0, width: 0, height: 0 })).toStrictEqual({
      ok: false,
      error: 'timeout',
    })

    expect(request.detachTab.mock.calls[0]?.[0]).toMatchObject({
      tab,
      x: 101,
      y: 200,
      width: 641,
      height: 480,
    })
  })
})
