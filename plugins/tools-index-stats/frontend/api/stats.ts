/**
 * The JSON messages of the index stats service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

export type { GetStatsRequest, Stats } from '../types'

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.index.stats'
