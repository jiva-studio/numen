/** The work the core reports to a window, in the windows' terms, and a cancel asked of it. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { TASK_KIND, TASK_UNIT, windowService } from '../clients'
import { createLostConnection, installCoreAnswers } from '../testing'
import { createTaskClient } from './tasks'

const request = { watchTasks: vi.fn(), cancelTask: vi.fn() }

const tasks = createTaskClient(windowService, 'desktop.example')

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ windowService: request })
})

afterEach(() => {
  restore()
})

const readAll = async <T>(stream: AsyncIterable<T>): Promise<T[]> => {
  const read: T[] = []
  for await (const one of stream) read.push(one)
  return read
}

describe('the work under way', () => {
  it('is counted in what each task counts, for the window named', async () => {
    request.watchTasks.mockImplementation(async function* () {
      yield {
        tasks: [
          {
            id: 'scan',
            title: 'Recognizing a scan',
            about: 'library/scan.pdf',
            done: 3n,
            total: 10n,
            unit: TASK_UNIT.BYTES,
            error: '',
            isUserStarted: true,
            pluginId: 'org.example.scans',
            pluginName: 'Scan reader',
            isCancellable: true,
            kind: TASK_KIND.NOTICE,
          },
          {
            id: 'later',
            title: 'Later',
            unit: 99,
            error: 'stopped',
            kind: TASK_KIND.UNSPECIFIED,
          },
        ],
      }
    })

    expect(await readAll(tasks.watchTasks(new AbortController().signal))).toStrictEqual([
      [
        {
          id: 'scan',
          label: 'Recognizing a scan',
          about: 'library/scan.pdf',
          progress: { completedCount: 3, total: 10, counting: 'bytes' },
          error: '',
          source: { isUserStarted: true, pluginId: 'org.example.scans', pluginName: 'Scan reader' },
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
    expect(request.watchTasks.mock.lastCall?.[0]).toMatchObject({ window: 'desktop.example' })
  })
})

describe('a cancel', () => {
  it('asks the core of the window to stop a task by its id', async () => {
    request.cancelTask.mockResolvedValue({})

    await tasks.cancelTask('making the vectors')

    expect(request.cancelTask.mock.lastCall?.[0]).toMatchObject({
      window: 'desktop.example',
      id: 'making the vectors',
    })
  })

  it('rejects when the core does not cancel the task', async () => {
    request.cancelTask.mockRejectedValue(createLostConnection('not found'))

    await expect(tasks.cancelTask('gone')).rejects.toThrow('not found')
  })
})
