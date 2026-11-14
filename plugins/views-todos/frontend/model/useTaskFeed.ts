import { ref, type Ref } from 'vue'
import type { TodosApi } from '../api/wire'
import type { Summary } from '../types'

/** The counts of the open tasks, kept up to date while the backend reports changes. */
export interface TaskFeed {
  /** The last counts read, or null before the first. */
  readonly summary: Readonly<Ref<Summary | null>>
  /** Follows the changes of the index and reads the counts at once. */
  start(): void
  /** Reads the counts again at the day getToday gives now. */
  refresh(): Promise<void>
  /** Ends the stream. */
  stop(): void
}

export function useTaskFeed(api: Pick<TodosApi, 'getSummary' | 'watchTasks'>, getToday: () => string): TaskFeed {
  return {
    summary: ref(null),
    start: () => undefined,
    refresh: () => Promise.resolve(),
    stop: () => undefined,
  }
}
