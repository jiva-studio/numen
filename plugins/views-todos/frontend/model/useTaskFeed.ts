import { ref, type Ref } from 'vue'
import type { TodosApi } from '../api/wire'
import type { Summary } from '../types'

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
): TaskFeed {
  const summary = ref<Summary | null>(null)
  let controller: AbortController | null = null

  const refresh = async (): Promise<void> => {
    const result = await api.getSummary(getToday())
    if (result.ok) summary.value = result.value
  }

  const follow = async (signal: AbortSignal): Promise<void> => {
    for await (const _message of api.watchTasks(signal)) {
      if (signal.aborted) return
      await refresh()
    }
  }

  return {
    summary,
    start: () => {
      if (controller) return
      controller = new AbortController()
      void follow(controller.signal)
    },
    refresh,
    stop: () => {
      controller?.abort()
      controller = null
    },
  }
}
