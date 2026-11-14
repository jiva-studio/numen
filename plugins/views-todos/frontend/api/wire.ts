/**
 * Wire adapter for the plugin's todos service: one function for each method.
 */
import { createApi, type ApiError, type Result } from '@numen/plugin-sdk'
import { SERVICE } from './todos'
import type {
  Filter,
  ListTasksRequest,
  ListTasksResponse,
  RefreshIndexRequest,
  RefreshIndexResponse,
  Rename,
  GetSummaryRequest,
  Summary,
  WatchMessage,
  WatchTasksRequest,
} from '../types'

const api = createApi(SERVICE)

/** What the models ask of the backend. */
export interface TodosApi {
  /** Reads the counts of the open tasks at a day. */
  getSummary(today: string): Promise<Result<Summary, ApiError>>
  /** Reads the tasks a filter accepts at a day, cut to a limit. */
  listTasks(today: string, filter: Filter, limit: number): Promise<Result<ListTasksResponse, ApiError>>
  /** Tells the backend which notes changed; no path and no rename is a full check. */
  refreshIndex(
    paths: readonly string[],
    renamed: readonly Rename[],
  ): Promise<Result<RefreshIndexResponse, ApiError>>
  /** Yields a message at once and after each change of the index, until the signal aborts. */
  watchTasks(signal: AbortSignal): AsyncGenerator<WatchMessage, ApiError | null>
}

export const todosApi: TodosApi = {
  getSummary: (today) => api.call<GetSummaryRequest, Summary>('GetSummary', { today }),
  listTasks: (today, filter, limit) =>
    api.call<ListTasksRequest, ListTasksResponse>('ListTasks', { today, filter, limit }),
  refreshIndex: (paths, renamed) =>
    api.call<RefreshIndexRequest, RefreshIndexResponse>('RefreshIndex', { paths, renamed }),
  watchTasks: (signal) => api.stream<WatchTasksRequest, WatchMessage>('WatchTasks', {}, { signal }),
}
