/**
 * A search asked as a person types: a pause after the last keystroke, and the
 * answer to the latest question alone.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createAnswerGuard, type AnswerGuard, type Question } from './questions'

export const SEARCH_DEBOUNCE_MS = 120

export type Wait = (ms: number) => Promise<unknown>

const sleep: Wait = (ms) => new Promise((wake) => setTimeout(wake, ms))

export interface DebouncedAskOptions {
  wait?: Wait
  guard?: AnswerGuard
}

export interface DebouncedAskSteps {
  blank(): void
  pending?(): void
  run(query: string, mine: Question): Promise<void>
}

export function useDebouncedAsk(options: DebouncedAskOptions = {}) {
  const wait = options.wait ?? sleep
  const guard = options.guard ?? createAnswerGuard()

  if (getCurrentScope()) onScopeDispose(guard.close)

  const ask = async (text: string, steps: DebouncedAskSteps): Promise<void> => {
    const mine = guard.ask()
    const query = text.trim()
    if (!query) return steps.blank()
    steps.pending?.()
    await wait(SEARCH_DEBOUNCE_MS)
    if (!mine.isCurrent) return
    await steps.run(query, mine)
  }

  return { ask, drop: guard.drop }
}

export function useSearchResults<T>(
  search: (query: string) => Promise<readonly T[]>,
  options: DebouncedAskOptions = {},
) {
  const results = shallowRef<readonly T[]>([])
  const isLoading = ref(false)
  const debounced = useDebouncedAsk(options)

  const setQuery = (text: string): Promise<void> =>
    debounced.ask(text, {
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
