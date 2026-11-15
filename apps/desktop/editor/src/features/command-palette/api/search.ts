/**
 * The two searches of the palette, asked of the core and read into the palette's own types.
 */
import { getNames, SEARCH_MODE, type SearchModeCode } from '@numen/core-client'
import type { Selector } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'
import { mapNoteType, mapRange } from '@/shared/api/wire'
import type { SearchDeps, SearchMode } from '../lib/search/types'
import type { JsonObject, JsonValue } from '@/shared/api/json'

const queryModes: Record<SearchModeCode, SearchMode | null> = {
  [SEARCH_MODE.UNSPECIFIED]: null,
  [SEARCH_MODE.HYBRID]: 'hybrid',
  [SEARCH_MODE.WORDS]: 'words',
  [SEARCH_MODE.MEANING]: 'meaning',
  [SEARCH_MODE.NAMES]: 'names',
}

const modes = getNames<SearchMode, SearchModeCode>(queryModes)

const parseSelector = (json: string): Selector | null => {
  try {
    const value: JsonValue = JSON.parse(json)
    return typeof value === 'object' && value !== null && !Array.isArray(value)
      ? (value as JsonObject)
      : null
  } catch {
    return null
  }
}

/** The searches of the palette over `search`. */
export const createSearchCore = (search: WindowClients['search']): SearchDeps => {
  const askCancellable = async <T>(
    signal: AbortSignal | undefined,
    ask: (searchId: string, options: { signal?: AbortSignal }) => Promise<T>,
  ): Promise<T> => {
    if (!signal) return ask('', {})
    const searchId = crypto.randomUUID()
    // An aborted fetch does not end the call in the core, so the abort cancels it by its id.
    const cancel = () => {
      search.cancelSearch({ searchId }).catch(() => undefined)
    }
    signal.addEventListener('abort', cancel, { once: true })
    try {
      return await ask(searchId, { signal })
    } finally {
      signal.removeEventListener('abort', cancel)
    }
  }

  return {
    names: async (query, limit, signal) => {
      const answer = await askCancellable(signal, (searchId, options) =>
        search.searchNames({ query, limit, searchId }, options),
      )
      return answer.found.map((one) => ({
        path: one.note?.path ?? '',
        title: one.note?.title ?? '',
        heading: one.heading?.text ?? '',
        line: one.heading?.line ?? -1,
        at: one.ranges.map(mapRange),
        type: mapNoteType(one.type),
        ...(one.tabKind ? { tabKind: one.tabKind } : {}),
      }))
    },
    search: async (query, mode, limit, signal) => {
      const answer = await askCancellable(signal, (searchId, options) =>
        search.searchPassages({ query, limit, mode: modes[mode], searchId }, options),
      )
      return answer.found.map((one) => ({
        path: one.path,
        title: one.note?.title ?? '',
        isNote: one.note !== undefined,
        type: mapNoteType(one.type),
        ...(one.tabKind ? { tabKind: one.tabKind } : {}),
        selector: parseSelector(one.selector),
        text: one.text,
        start: one.range?.start ?? 0,
        length: (one.range?.end ?? 0) - (one.range?.start ?? 0),
        line: one.line,
        at: one.ranges.map(mapRange),
      }))
    },
  }
}
