/**
 * The JSON messages of the todos service the plugin's backend serves. Keys and meaning match the Go
 * messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */
import type {
  GetSummaryRequest,
  ListTasksRequest,
  ListTasksResponse,
  RefreshIndexRequest,
  RefreshIndexResponse,
  Summary,
  WatchMessage,
  WatchTasksRequest,
} from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.todos'

/** The request and response of each method by name. */
export interface Methods {
  GetSummary: { request: GetSummaryRequest; response: Summary }
  ListTasks: { request: ListTasksRequest; response: ListTasksResponse }
  RefreshIndex: { request: RefreshIndexRequest; response: RefreshIndexResponse }
  WatchTasks: { request: WatchTasksRequest; response: WatchMessage }
}
