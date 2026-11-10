/**
 * The JSON messages of the index stats service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

import type { GetStatsRequest, Stats } from '../types'

export type { GetStatsRequest, Stats } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.index.stats'

/** The request and response of each method by name. */
export interface Methods {
  GetStats: { request: GetStatsRequest; response: Stats }
}
