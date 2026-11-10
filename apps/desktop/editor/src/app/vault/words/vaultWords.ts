/**
 * Search mode conversions between schema representations and window models.
 */
import { getNames, SEARCH_MODE, type SearchModeCode } from '@numen/core-client'
import type { SearchMode } from '@/features/command-palette'

/**
 * Search modes, named as the window uses them.
 */
const query: Record<SearchModeCode, SearchMode | null> = {
  [SEARCH_MODE.UNSPECIFIED]: null,
  [SEARCH_MODE.HYBRID]: 'hybrid',
  [SEARCH_MODE.WORDS]: 'words',
  [SEARCH_MODE.MEANING]: 'meaning',
  [SEARCH_MODE.NAMES]: 'names',
}

/** Search modes, as the schema names them. */
export const modes = getNames<SearchMode, SearchModeCode>(query)

