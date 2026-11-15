/**
 * One saved search tab: it follows the watch of the note, reads the cached results on each change of
 * it, opens a result at its range and saves the query of the note. It never asks for a run.
 */
import { readonly, ref, shallowRef, type Ref } from 'vue'
import type { Caught, TextRange } from '@numen/plugin-sdk'
import { WORDS as words } from '../../words'
import type { SavedSearchService } from '../api/service'
import type { GetResultsResponse } from '../api/types'
import { formatProblem, getResultsView, type ResultsView } from '../lib/view'

/** What the window gives the saved search tabs. */
export interface SavedSearchTabDeps {
  readonly service: SavedSearchService
  readonly openAt: (path: string, ranges: readonly TextRange[]) => void
  readonly locale: string
}

/** The query, the rerank text and the limit the person saves; an empty rerank and a limit of 0 remove them. */
export interface SearchChange {
  readonly query: string
  readonly rerank: string
  readonly limit: number
}

/**
 * What SavedSearchTab draws and the actions it calls. view is null until the first GetResults answers;
 * failure is '' when nothing failed; saveProblems are those of the last save, as line:column and message.
 */
export interface SavedSearchTabState {
  readonly path: Readonly<Ref<string>>
  readonly view: Readonly<Ref<ResultsView | null>>
  readonly failure: Readonly<Ref<string>>
  readonly isSaving: Readonly<Ref<boolean>>
  readonly isEditing: Readonly<Ref<boolean>>
  readonly saveProblems: Readonly<Ref<readonly string[]>>
  readonly actions: {
    openResult(index: number): void
    saveSearch(change: SearchChange): Promise<void>
    setEditing(isEditing: boolean): void
  }
}

/** One tab and what starts and ends its following. */
export interface SavedSearchTab {
  readonly state: SavedSearchTabState
  /** Starts following the note, unless a watch of it runs. */
  follow(): void
  /** Ends the watch and every call, for good. */
  dispose(): void
}

const getMessage = (error: Caught): string =>
  error instanceof Error ? error.message : String(error)

/** The tab reads GetResults on each change of the watch, the last change winning, and runs no search. */
export function createSavedSearchTab(notePath: string, deps: SavedSearchTabDeps): SavedSearchTab {
  const { service } = deps
  const path = ref(notePath)
  const response = shallowRef<GetResultsResponse | null>(null)
  const view = shallowRef<ResultsView | null>(null)
  const failure = ref('')
  const isSaving = ref(false)
  const isEditing = ref(false)
  const saveProblems = shallowRef<readonly string[]>([])
  const life = new AbortController()
  let watching: AbortController | null = null
  let reading: AbortController | null = null

  const show = (answer: GetResultsResponse): void => {
    response.value = answer
    view.value = getResultsView(answer, words, deps.locale)
    failure.value = ''
  }

  const read = (): Promise<void> => {
    reading?.abort()
    const call = new AbortController()
    reading = call
    return service.getResults({ path: path.value }, call.signal).then(
      (answer) => {
        if (!call.signal.aborted) show(answer)
      },
      (error: Caught) => {
        if (!call.signal.aborted) failure.value = getMessage(error)
      },
    )
  }

  const watch = async (call: AbortController): Promise<void> => {
    try {
      for await (const _change of service.watchResults({ path: path.value }, call.signal)) {
        read().catch(() => undefined)
      }
    } catch (error) {
      if (!call.signal.aborted) failure.value = getMessage(error)
    } finally {
      if (watching === call) watching = null
    }
  }

  const follow = (): void => {
    if (watching || life.signal.aborted) return
    const call = new AbortController()
    watching = call
    watch(call).catch(() => undefined)
  }

  const openResult = (index: number): void => {
    const result = response.value?.results[index]
    if (!result) return
    deps.openAt(result.path, [{ start: result.start, end: result.start + result.length }])
  }

  const saveSearch = async (change: SearchChange): Promise<void> => {
    isSaving.value = true
    try {
      const answer = await service.updateSearch({ path: path.value, ...change }, life.signal)
      saveProblems.value = answer.errors.map(formatProblem)
      if (answer.errors.length === 0) {
        isEditing.value = false
        await read()
      }
    } catch (error) {
      if (!life.signal.aborted) failure.value = getMessage(error)
    } finally {
      isSaving.value = false
    }
  }

  const dispose = (): void => {
    life.abort()
    watching?.abort()
    reading?.abort()
  }

  return {
    state: {
      path: readonly(path),
      view: readonly(view),
      failure: readonly(failure),
      isSaving: readonly(isSaving),
      isEditing: readonly(isEditing),
      saveProblems: readonly(saveProblems),
      actions: {
        openResult,
        saveSearch,
        setEditing: (next: boolean) => {
          isEditing.value = next
        },
      },
    },
    follow,
    dispose,
  }
}
