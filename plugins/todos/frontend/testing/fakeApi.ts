/**
 * A TodosApi for the tests: it answers from values the test sets, records each call, and streams the
 * messages the test pushes. It is the outer edge of the frontend; the models run on it for real.
 */
import type { ApiError, Result } from '@numen/plugin-sdk'
import type { TodosApi } from '../api/wire'
import type {
  Filter,
  ListTasksResponse,
  Rename,
  SetTaskDoneRequest,
  SetTaskDoneResponse,
  Summary,
  Task,
  WatchMessage,
} from '../types'

export interface ListCall {
  readonly today: string
  readonly filter: Filter
  readonly limit: number
}

export interface RefreshCall {
  readonly paths: readonly string[]
  readonly renamed: readonly Rename[]
}

export interface FakeCalls {
  readonly getSummary: string[]
  readonly listTasks: ListCall[]
  readonly refreshIndex: RefreshCall[]
  readonly setTaskDone: SetTaskDoneRequest[]
  watchStarts: number
  watchEnds: number
}

export const SUMMARY: Summary = {
  isReady: true,
  overdueCount: 2,
  todayCount: 3,
  noDateCount: 1,
  unreadableCount: 0,
}

const ok = <T>(value: T): Result<T, ApiError> => ({ ok: true, value })

export const task = (overrides: Partial<Task> & Pick<Task, 'path' | 'line' | 'text'>): Task => ({
  raw: `- [ ] ${overrides.text}`,
  status: 'open',
  priority: '',
  due: '',
  doneOn: '',
  recurrence: '',
  projects: [],
  contexts: [],
  issues: [],
  depth: 0,
  ...overrides,
})

/**
 * The tasks the fake answers a ListTasks with are keyed by the due filter of the request. A
 * SetTaskDone answers with setTaskDone when it is set; else it ticks the task as the backend does:
 * the task is removed from the lists and returned with the status done.
 */
export interface FakeInit {
  readonly summary?: Summary
  readonly tasks?: Partial<Record<Filter['due'], ListTasksResponse>>
  readonly setTaskDone?: (request: SetTaskDoneRequest) => Result<SetTaskDoneResponse, ApiError>
}

/** A SetTaskDone answer that refuses the change. */
export const refuse = (
  request: SetTaskDoneRequest,
  error: SetTaskDoneResponse['error'],
): Result<SetTaskDoneResponse, ApiError> =>
  ok({ task: task({ path: request.path, line: request.line, text: '' }), change: '', error })

export function createFakeApi(init: FakeInit = {}) {
  let summary = init.summary ?? SUMMARY
  let lists: Partial<Record<Filter['due'], ListTasksResponse>> = { ...init.tasks }
  const queue: WatchMessage[] = []
  let wake: (() => void) | null = null
  let isEnding = false
  const calls: FakeCalls = {
    getSummary: [],
    listTasks: [],
    refreshIndex: [],
    setTaskDone: [],
    watchStarts: 0,
    watchEnds: 0,
  }

  const api: TodosApi = {
    getSummary: async (today) => {
      calls.getSummary.push(today)
      return ok(summary)
    },
    listTasks: async (today, filter, limit) => {
      calls.listTasks.push({ today, filter, limit })
      return ok(lists[filter.due] ?? { tasks: [], total: 0, revision: 1 })
    },
    refreshIndex: async (paths, renamed) => {
      calls.refreshIndex.push({ paths, renamed })
      return ok({})
    },
    setTaskDone: async (request) => {
      calls.setTaskDone.push(request)
      if (init.setTaskDone) return init.setTaskDone(request)
      let ticked: Task | null = null
      for (const [due, list] of Object.entries(lists) as [Filter['due'], ListTasksResponse][]) {
        const kept = list.tasks.filter(
          (one) => one.path !== request.path || one.line !== request.line,
        )
        if (kept.length === list.tasks.length) continue
        ticked = list.tasks.find((one) => !kept.includes(one)) ?? null
        lists[due] = { tasks: kept, total: list.total - 1, revision: list.revision + 1 }
      }
      if (!ticked) return refuse(request, 'notFound')
      return ok({
        task: { ...ticked, status: 'done', doneOn: request.today },
        change: 'completed',
        error: '',
      })
    },
    watchTasks: async function* (signal) {
      calls.watchStarts += 1
      while (!signal.aborted && !isEnding) {
        const next = queue.shift()
        if (next) {
          yield next
          continue
        }
        await new Promise<void>((resolve) => {
          wake = resolve
          signal.addEventListener('abort', () => resolve(), { once: true })
        })
      }
      isEnding = false
      calls.watchEnds += 1
      return null
    },
  }

  return {
    api,
    calls,
    setSummary: (next: Summary) => {
      summary = next
    },
    setTasks: (next: FakeInit['tasks']) => {
      lists = { ...next }
    },
    /** Sends a message down the stream. */
    push: (message: WatchMessage) => {
      queue.push(message)
      wake?.()
    },
    /** Ends the stream the way a backend that stops does. */
    end: () => {
      isEnding = true
      wake?.()
    },
  }
}

/** A timer the test runs by hand. */
export function createFakeTimer() {
  const waits: { run: () => void; delayMs: number; handle: number }[] = []
  let next = 1
  return {
    waits,
    timer: {
      set: (run: () => void, delayMs: number) => {
        const handle = next++
        waits.push({ run, delayMs, handle })
        return handle
      },
      clear: (handle: number) => {
        const index = waits.findIndex((one) => one.handle === handle)
        if (index !== -1) waits.splice(index, 1)
      },
    },
    /** Runs the oldest wait that is still set. */
    fire: () => {
      const one = waits.shift()
      one?.run()
    },
  }
}

/** Lets the promises that are ready run. */
export const settle = async (): Promise<void> => {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}
