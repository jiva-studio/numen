import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { installCoreAnswers } from '@numen/core-client/testing'
import { watchTasks } from './tasks'

const core = { watchTasks: vi.fn() }

let restore: () => void

beforeEach(() => {
  vi.resetAllMocks()
  restore = installCoreAnswers({ windowService: core })
})

afterEach(() => {
  restore()
})

describe('the tasks of the window', () => {
  it('are followed for the target of this window and read as notice tasks', async () => {
    core.watchTasks.mockImplementation(async function* () {
      yield {
        tasks: [
          {
            id: 'indexing\t01A',
            title: 'Indexing the vault',
            about: 'Garden',
            error: '',
            isUserStarted: false,
          },
        ],
      }
    })

    const lists: unknown[] = []
    for await (const tasks of watchTasks(new AbortController().signal)) lists.push(tasks)

    expect(core.watchTasks.mock.calls[0]?.[0]).toMatchObject({ window: 'desktop.flashcards' })
    expect(lists).toStrictEqual([
      [
        {
          id: 'indexing\t01A',
          label: 'Indexing the vault',
          about: 'Garden',
          error: '',
          isUserStarted: false,
        },
      ],
    ])
  })
})
