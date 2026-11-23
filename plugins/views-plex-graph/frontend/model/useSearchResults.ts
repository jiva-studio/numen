/**
 * A search performed as the user types: a pause after the last keystroke, and the
 * response to the latest query only.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createRequestGuard, type RequestGuard, type InFlightRequest } from '@numen/ui'
import { SEARCH_DEBOUNCE_MS } from './searchDebounce'

type Wait = (ms: number) => Promise<void>

const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

export interface DebouncedSearchOptions {
  wait?: Wait
  guard?: RequestGuard
}

interface DebouncedSearchSteps {
  showBlank(): void
  showPending?(): void
  run(query: string, mine: InFlightRequest): Promise<void>
}

function useDebouncedSearch(options: DebouncedSearchOptions = {}) {
  const wait = options.wait ?? sleep
  const guard = options.guard ?? createRequestGuard()

  if (getCurrentScope()) onScopeDispose(guard.close)

  const search = async (text: string, steps: DebouncedSearchSteps): Promise<void> => {
    const mine = guard.acquireRequest()
    const query = text.trim()
    if (!query) return steps.showBlank()
    steps.showPending?.()
    await wait(SEARCH_DEBOUNCE_MS)
    if (!mine.isCurrent) return
    await steps.run(query, mine)
  }

  return { search, ask: search, cancel: guard.cancel, drop: guard.cancel }
}

export function useSearchResults<T>(
  getSearch: () => (query: string) => Promise<readonly T[]>,
  options: DebouncedSearchOptions = {},
) {
  const results = shallowRef<readonly T[]>([])
  const isLoading = ref(false)
  const debounced = useDebouncedSearch(options)

  const setQuery = (text: string): Promise<void> =>
    debounced.search(text, {
      showBlank: () => {
        results.value = []
        isLoading.value = false
      },
      showPending: () => {
        isLoading.value = true
      },
      run: async (query, mine) => {
        try {
          const found = await getSearch()(query)
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
