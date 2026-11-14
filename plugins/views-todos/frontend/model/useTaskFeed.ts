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
  _api: Pick<TodosApi, 'getSummary' | 'watchTasks'>,
  _getToday: () => string,
): TaskFeed {
  return {
    summary: ref(null),
    start: () => undefined,
    refresh: () => Promise.resolve(),
    stop: () => undefined,
  }
}
