/**
 * A search run while the user types: a pause after the last keystroke, and only
 * the response to the latest query is used.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createRequestGuard, type RequestGuard, type InFlightRequest } from './requestGuard'

/** The delay after a keystroke before the search runs. */
export const SEARCH_DEBOUNCE_MS = 120

/** Resolves after the given milliseconds. */
export type Wait = (ms: number) => Promise<unknown>

const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

export interface DebouncedSearchOptions {
  wait?: Wait
  /** The guard the requests are measured against. A caller asking other things too hands in its own. */
  guard?: RequestGuard
}

/** What a typed text evaluates to. */
export interface DebouncedSearchSteps {
  /** The text holds nothing to ask. */
  blank(): void
  /** The text will be searched once the pause ends. */
  pending?(): void
  /** The pause ended with this text still the latest. */
  run(query: string, mine: InFlightRequest): Promise<void>
}

export function useDebouncedSearch(options: DebouncedSearchOptions = {}) {
  const wait = options.wait ?? sleep
  const guard = options.guard ?? createRequestGuard()

  if (getCurrentScope()) onScopeDispose(guard.close)

  const search = async (text: string, steps: DebouncedSearchSteps): Promise<void> => {
    const mine = guard.acquireRequest()
    const query = text.trim()
    if (!query) return steps.blank()
    steps.pending?.()
    await wait(SEARCH_DEBOUNCE_MS)
    if (!mine.isCurrent) return
    await steps.run(query, mine)
  }

  return { search, ask: search, cancel: guard.cancel, drop: guard.cancel }
}

/** The results of a search run while the user types, and whether one is in progress. */
export function useSearchResults<T>(
  search: (query: string) => Promise<readonly T[]>,
  options: DebouncedSearchOptions = {},
) {
  const results = shallowRef<readonly T[]>([])
  const isLoading = ref(false)
  const debounced = useDebouncedSearch(options)

  const setQuery = (text: string): Promise<void> =>
    debounced.search(text, {
      blank: () => {
        results.value = []
        isLoading.value = false
      },
      pending: () => {
        isLoading.value = true
      },
      run: async (query, mine) => {
        try {
          const found = await search(query)
          if (mine.isCurrent) results.value = found
        } catch {
          // A failed search yields no results, and typing continues.
          if (mine.isCurrent) results.value = []
        } finally {
          if (mine.isCurrent) isLoading.value = false
        }
      },
    })

  return { results, isLoading, setQuery }
}
