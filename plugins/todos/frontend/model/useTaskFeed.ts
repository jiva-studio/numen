import { ref, type Ref } from 'vue'
import type { TodosApi } from '../api/wire'
import type { Summary } from '../types'
import type { Timer } from './timer'

/** How long the feed waits before it opens the stream again after the stream ended, in milliseconds. */
export const RECONNECT_DELAY_MS = 1000

/** The counts of the open tasks, kept up to date while the backend reports changes. The summary is null before the first read. */
export interface TaskFeed {
  readonly summary: Readonly<Ref<Summary | null>>
  /** Follows the changes of the index and reads the counts at once. */
  start(): void
  /** Reads the counts again at the day getToday gives now. */
  refresh(): Promise<void>
  /** Ends the stream. */
  stop(): void
}

export function useTaskFeed(
  api: Pick<TodosApi, 'getSummary' | 'watchTasks'>,
  getToday: () => string,
  timer: Timer,
): TaskFeed {
  const summary = ref<Summary | null>(null)
  let controller: AbortController | null = null
  let retry: number | null = null
  let latest = 0

  // Only the answer to the last read is kept: an earlier read that answers later holds older counts.
  const refresh = async (): Promise<void> => {
    latest += 1
    const asked = latest
    const result = await api.getSummary(getToday())
    if (result.ok && asked === latest) summary.value = result.value
  }

  const follow = async (signal: AbortSignal): Promise<void> => {
    for await (const _message of api.watchTasks(signal)) {
      if (signal.aborted) return
      await refresh()
    }
  }

  const connect = (): void => {
    retry = null
    const current = controller
    if (!current) return
    const reconnect = (): void => {
      if (controller === current && !current.signal.aborted) {
        retry = timer.set(connect, RECONNECT_DELAY_MS)
      }
    }
    follow(current.signal).then(reconnect, reconnect)
  }

  return {
    summary,
    start: () => {
      if (controller) return
      controller = new AbortController()
      connect()
    },
    refresh,
    stop: () => {
      if (retry !== null) timer.clear(retry)
      retry = null
      controller?.abort()
      controller = null
    },
  }
}
