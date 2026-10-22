/**
 * A search performed as a person types: a pause after the last keystroke, and the
 * response to the latest query alone.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createRequestGuard, type RequestGuard, type InFlightRequest } from '@numen/plugin-sdk'

/** How long a keystroke waits before anything is asked. */
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
  /** The text will be asked once the pause ends. */
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

/** The answers to a search as a person types, and whether one is on its way. */
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
          // A search that fails offers nothing, and the person goes on typing.
          if (mine.isCurrent) results.value = []
        } finally {
          if (mine.isCurrent) isLoading.value = false
        }
      },
    })

  return { results, isLoading, setQuery }
}
