/**
 * A search performed as a person types: a pause after the last keystroke, and the
 * response to the latest query alone.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createRequestGuard, type RequestGuard, type InFlightRequest } from '../lib/requestGuard'

export const SEARCH_DEBOUNCE_MS = 120

export type Wait = (ms: number) => Promise<unknown>

const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

export interface DebouncedSearchOptions {
  wait?: Wait
  guard?: RequestGuard
}

export interface DebouncedSearchSteps {
  blank(): void
  pending?(): void
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
          if (mine.isCurrent) results.value = []
        } finally {
          if (mine.isCurrent) isLoading.value = false
        }
      },
    })

  return { results, isLoading, setQuery }
}
