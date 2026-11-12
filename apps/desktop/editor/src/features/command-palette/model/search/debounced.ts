/**
 * A search run while the user types: a pause after the last keystroke, and only
 * the response to the latest query is used.
 */
import { getCurrentScope, onScopeDispose } from 'vue'
import { createRequestGuard, type RequestGuard, type InFlightRequest } from '@numen/ui'

/** The delay after a keystroke before the search runs. */
const SEARCH_DEBOUNCE_MS = 120

/** Resolves after the given milliseconds. */
export type Wait = (ms: number) => Promise<void>

const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

export interface DebouncedSearchOptions {
  wait?: Wait
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

  // Stryker disable next-line ConditionalExpression: guards onScopeDispose when called outside an active effectScope
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
