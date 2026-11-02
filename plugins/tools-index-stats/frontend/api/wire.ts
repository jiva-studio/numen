/**
 * Wire adapter for the plugin's index stats service.
 */
import { createApi, type ApiError, type Result } from '@numen/plugin-sdk'
import { SERVICE, type GetStatsRequest, type Stats } from './stats'

const api = createApi(SERVICE)

/** What the model asks of the backend. */
export interface StatsApi {
  /** Reads what the search index holds now. */
  get(): Promise<Result<Stats, ApiError>>
}

export const statsApi: StatsApi = {
  get: () => api.call<GetStatsRequest, Stats>('GetStats', {}),
}
