/**
 * The requests and streams of the window to the core, sent for this window.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { FLUSH_RESULT, TASK_KIND, TASK_UNIT } from '@numen/core-client'
import { createLostConnection } from '@numen/core-client/testing'
import { createHandledClients } from '@/testing/coreAnswers'
import { createTaskApi } from './tasks'
import { createQuitApi } from './quit'
import { createFocusApi } from './focus'
import { createShownVaultApi } from './shownVault'

const request = {
  reportFlush: vi.fn(),
  watchFocus: vi.fn(),
  watchTasks: vi.fn(),
  cancelTask: vi.fn(),
  watchQuit: vi.fn(),
  getCurrentVault: vi.fn(),
}

const clients = createHandledClients(request)
const api = {
  ...createTaskApi(clients.windowService),
  ...createQuitApi(clients.windowService),
  ...createFocusApi(clients.workspace),
  ...createShownVaultApi(clients.windowService),
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('a flush reported', () => {
  it('says nothing was left where the window had nothing unsaved', async () => {
    await api.reportFlush('token-1')

    expect(request.reportFlush.mock.lastCall?.[0]).toMatchObject({
      window: 'editor',
      token: 'token-1',
      result: FLUSH_RESULT.NOTHING,
    })
  })

  it('says what the window wrote', async () => {
    await api.reportFlush('token-1', 'written')

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

describe('the vault the window shows', () => {
  it('is the vault the core names for this window', async () => {
    request.getCurrentVault.mockResolvedValue({ vault: 'physics' })

    expect(await api.getShownVault()).toBe('physics')
    expect(request.getCurrentVault.mock.lastCall?.[0]).toMatchObject({ window: 'editor' })
  })
})

describe('the streams a window follows', () => {
  const signal = new AbortController().signal

  it('follows the focus', async () => {
    request.watchFocus.mockImplementation(async function* () {
      yield { path: 'Heat.md', ranges: [{ start: 4, end: 9 }], tabId: 'note:1' }
    })

    expect(await readAll(api.watchFocus(signal))).toStrictEqual([
      { path: 'Heat.md', ranges: [{ start: 4, end: 9 }], tabId: 'note:1' },
    ])
  })

  it('releases the focus on the signal', async () => {
    let requestSignal: AbortSignal | undefined
    request.watchFocus.mockImplementation(async function* (_request, context) {
      requestSignal = context.signal
      await new Promise((resolve) => context.signal.addEventListener('abort', resolve))
      yield* [] as object[]
    })
    const controller = new AbortController()

    const readTask = readAll(api.watchFocus(controller.signal))
    await vi.waitFor(() => expect(requestSignal).toBeDefined())
    controller.abort()

    await vi.waitFor(() => expect(requestSignal?.aborted).toBe(true))
    await readTask.catch(() => undefined)
  })

  it('follows a quit requested of this window', async () => {
    request.watchQuit.mockImplementation(async function* () {
      yield { token: 'token-9', flush: true }
    })

    expect(await readAll(api.watchQuit(signal))).toStrictEqual([{ token: 'token-9', flush: true }])
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

    expect(await readAll(api.watchTasks(signal))).toStrictEqual([
      [
        {
          id: 'scan',
          label: 'Recognising a scan',
          about: 'library/scan.pdf',
          progress: { completedCount: 3, total: 10, counting: 'bytes' },
          error: '',
          source: { isUserStarted: true, pluginId: 'md.example.scans', pluginName: 'Scan reader' },
          isCancellable: true,
          kind: 'notice',
        },
        {
          id: 'later',
          label: 'Later',
          about: '',
          progress: { completedCount: 0, total: 0, counting: 'items' },
          error: 'stopped',
          source: { isUserStarted: false, pluginId: '', pluginName: '' },
          isCancellable: false,
          kind: 'work',
        },
      ],
    ])
    expect(request.watchTasks.mock.lastCall?.[0]).toMatchObject({ window: 'editor' })
  })

  it('asks the core of this window to cancel a task by its id', async () => {
    request.cancelTask.mockResolvedValue({})

    await api.cancelTask('making the vectors')

    expect(request.cancelTask.mock.lastCall?.[0]).toMatchObject({
      window: 'editor',
      id: 'making the vectors',
    })
  })

  it('rejects when the core does not cancel the task', async () => {
    request.cancelTask.mockRejectedValue(createLostConnection('not found'))

    await expect(api.cancelTask('gone')).rejects.toThrow('not found')
  })
})
