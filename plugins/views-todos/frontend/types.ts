/**
 * What the frontend reads of the backend: the messages of the todos service. Keys and meaning match
 * the Go messages in backend/api.
 */

/** What the character of a checkbox says: a space is open, x is done, any other character is other. */
export type Status = 'open' | 'done' | 'other'

/** A field of a task line that is written but cannot be read. */
export type Issue = 'invalidDue' | 'invalidDone' | 'invalidRecurrence' | 'invalidPriority'

/** One task line of a note. Dates are `YYYY-MM-DD` or empty. */
export interface Task {
  readonly path: string
  readonly line: number
  readonly raw: string
  readonly text: string
  readonly status: Status
  readonly priority: string
  readonly due: string
  readonly doneOn: string
  readonly recurrence: string
  readonly projects: readonly string[]
  readonly contexts: readonly string[]
  readonly issues: readonly Issue[]
  readonly depth: number
}

export type DueFilter = 'any' | 'none' | 'overdue' | 'today' | 'thisWeek' | 'range'

/** What ListTasks selects by; its fields combine with AND. */
export interface Filter {
  readonly status: readonly Status[]
  readonly due: DueFilter
  readonly from: string
  readonly to: string
  readonly projects: readonly string[]
  readonly contexts: readonly string[]
  readonly priorities: readonly string[]
  readonly path: string
  readonly text: string
}

export interface GetSummaryRequest {
  readonly today: string
}

/** The counts of the open tasks. */
export interface Summary {
  readonly isReady: boolean
  readonly overdueCount: number
  readonly todayCount: number
  readonly noDateCount: number
  readonly unreadableCount: number
}

export interface ListTasksRequest {
  readonly today: string
  readonly filter: Filter
  readonly limit: number
}

/** The total is the count before the limit. */
export interface ListTasksResponse {
  readonly tasks: readonly Task[]
  readonly total: number
  readonly revision: number
}

export interface Rename {
  readonly from: string
  readonly to: string
}

/** No path and no rename is a full check. */
export interface RefreshIndexRequest {
  readonly paths: readonly string[]
  readonly renamed: readonly Rename[]
}

export type RefreshIndexResponse = Record<string, never>

export type WatchTasksRequest = Record<string, never>

/** Sent once at the start of the stream and after each change of the index. */
export interface WatchMessage {
  readonly revision: number
  readonly isReady: boolean
}
