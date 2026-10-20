/**
 * A search asked as a person types: a pause after the last keystroke, and the
 * answer to the latest question alone.
 */
import { getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { createAnswerGuard, type AnswerGuard, type Question } from '@numen/plugin-sdk'

/** How long a keystroke waits before anything is asked. */
export const SEARCH_DEBOUNCE_MS = 120

/** Resolves after the given milliseconds. */
export type Wait = (ms: number) => Promise<unknown>

const sleep: Wait = (ms) => new Promise((wake) => setTimeout(wake, ms))

export interface DebouncedAskOptions {
  wait?: Wait
  /** The guard the questions are measured against. A caller asking other things too hands in its own. */
  guard?: AnswerGuard
}

/** What a typed text comes to. */
export interface DebouncedAskSteps {
  /** The text holds nothing to ask. */
  blank(): void
  /** The text will be asked once the pause ends. */
  pending?(): void
  /** The pause ended with this text still the latest. */
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

/** The answers to a search as a person types, and whether one is on its way. */
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
          // A search that fails offers nothing, and the person goes on typing.
          if (mine.isCurrent) results.value = []
        } finally {
          if (mine.isCurrent) isLoading.value = false
        }
      },
    })

  return { results, isLoading, setQuery }
}
