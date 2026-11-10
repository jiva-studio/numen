/**
 * Search mode and model presence conversions between schema representations and window models.
 */
import {
  getNames,
  MODEL_PRESENCE,
  SEARCH_MODE,
  type ModelPresenceCode,
  type SearchModeCode,
} from '@numen/core-client'
import type { SearchMode } from '@/features/command-palette'
import type { Presence } from '@/entities/settings'

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

/** What a model's files are on this machine, in the words the window uses. */
export const fetched: Record<ModelPresenceCode, Presence> = {
  [MODEL_PRESENCE.UNSPECIFIED]: 'nothing to fetch',
  [MODEL_PRESENCE.PRESENT]: 'present',
  [MODEL_PRESENCE.NOT_FETCHED]: 'not fetched',
  [MODEL_PRESENCE.NOTHING_TO_FETCH]: 'nothing to fetch',
}
