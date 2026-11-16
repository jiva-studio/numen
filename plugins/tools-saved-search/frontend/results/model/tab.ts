/**
 * One saved search tab: it follows the watch of the note, reads the cached results on each change of
 * it, filters them by a text and by their sources, opens a result at its range, tells of a failed run
 * in a notice and saves the query of the note. It never asks for a run.
 */
import { computed, readonly, ref, shallowRef, type Ref } from 'vue'
import type { Caught, TextRange } from '@numen/plugin-sdk'
import { WORDS as words } from '../../words'
import type { SavedSearchService } from '../api/service'
import type { GetResultsResponse, Problem, SearchFunction } from '../api/types'
import {
  filterItems,
  isInQuery,
  getResultsView,
  type ResultItemView,
  type ResultsView,
} from '../lib/view'

/** What the window gives the saved search tabs. */
export interface SavedSearchTabDeps {
  readonly service: SavedSearchService
  readonly openAt: (path: string, ranges: readonly TextRange[]) => void
  readonly notify: (text: string) => void
  readonly locale: string
  readonly getNow: () => number
}

/** The query the person saves. */
export interface SearchChange {
  readonly query: string
}

/**
 * What SavedSearchTab draws and the actions it calls. view is null until the first GetResults answers;
 * items are the results the filter and the hidden sources leave; failure is '' when nothing failed;
 * queryProblems are those the editor underlines, at their line and column in the query: those of the
 * last save when it had any, else those of the saved query that have a place in it; functions is the
 * catalog of the search language, read when the editor first opens; getNow gives the time of the
 * wall clock in milliseconds, which the times of the run are written from.
 */
export interface SavedSearchTabState {
  readonly path: Readonly<Ref<string>>
  readonly view: Readonly<Ref<ResultsView | null>>
  readonly items: Readonly<Ref<readonly ResultItemView[]>>
  readonly filter: Readonly<Ref<string>>
  readonly hiddenSources: Readonly<Ref<ReadonlySet<string>>>
  readonly failure: Readonly<Ref<string>>
  readonly isEditing: Readonly<Ref<boolean>>
  readonly queryProblems: Readonly<Ref<readonly Problem[]>>
  readonly functions: Readonly<Ref<readonly SearchFunction[]>>
  readonly getNow: () => number
  readonly actions: {
    openResult(index: number): void
    saveSearch(change: SearchChange): Promise<void>
    setEditing(isEditing: boolean): void
    setFilter(filter: string): void
    setSourceShown(path: string, isShown: boolean): void
    showAllSources(): void
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

const getFailureKey = (answer: GetResultsResponse): string =>
  answer.failedAt === '' ? '' : `${answer.failedAt}\n${answer.failure}`

const getNotice = (answer: GetResultsResponse): string => `${answer.title}: ${answer.failure}`

/** The tab reads GetResults on each change of the watch, the last change winning, and runs no search. */
export function createSavedSearchTab(notePath: string, deps: SavedSearchTabDeps): SavedSearchTab {
  const { service } = deps
  const path = ref(notePath)
  const response = shallowRef<GetResultsResponse | null>(null)
  const view = shallowRef<ResultsView | null>(null)
  const filter = ref('')
  const hiddenSources = shallowRef<ReadonlySet<string>>(new Set())
  const failure = ref('')
  const isEditing = ref(false)
  const saveProblems = shallowRef<readonly Problem[]>([])
  const functions = shallowRef<readonly SearchFunction[]>([])
  const isAnswerObsolete = ref(false)
  let functionsRead: Promise<void> | null = null
  let catalogFailure: string | null = null
  let editorSession = 0
  const life = new AbortController()
  let watching: AbortController | null = null
  let reading: AbortController | null = null
  let shownFailure: string | null = null

  const items = computed(() =>
    filterItems(view.value?.items ?? [], filter.value, hiddenSources.value),
  )

  const queryProblems = computed<readonly Problem[]>(() => {
    if (saveProblems.value.length > 0) return saveProblems.value
    if (isAnswerObsolete.value) return []
    return (response.value?.errors ?? []).filter(isInQuery)
  })

  const tellFailure = (answer: GetResultsResponse): void => {
    const key = getFailureKey(answer)
    if (shownFailure !== null && key !== '' && key !== shownFailure) deps.notify(getNotice(answer))
    shownFailure = key
  }

  const show = (answer: GetResultsResponse): void => {
    tellFailure(answer)
    response.value = answer
    isAnswerObsolete.value = false
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
    if (result.isWholeFile) {
      deps.openAt(result.path, [{ start: 0, end: 0 }])
      return
    }
    deps.openAt(result.path, [{ start: result.start, end: result.start + result.length }])
  }

  const saveSearch = async (change: SearchChange): Promise<void> => {
    const session = editorSession
    try {
      const answer = await service.updateSearch({ path: path.value, ...change }, life.signal)
      if (session === editorSession) saveProblems.value = answer.errors
      if (answer.errors.length === 0) {
        isAnswerObsolete.value = true
        await read()
      }
    } catch (error) {
      if (!life.signal.aborted) failure.value = getMessage(error)
    }
  }

  const readFunctions = (): void => {
    functionsRead ??= service.listSearchFunctions({}, life.signal).then(
      (answer) => {
        functions.value = answer.functions
        if (catalogFailure !== null && failure.value === catalogFailure) failure.value = ''
        catalogFailure = null
      },
      (error: Caught) => {
        functionsRead = null
        if (life.signal.aborted) return
        catalogFailure = getMessage(error)
        failure.value = catalogFailure
      },
    )
  }

  const setEditing = (next: boolean): void => {
    isEditing.value = next
    editorSession++
    if (next) readFunctions()
    else saveProblems.value = []
  }

  const setSourceShown = (source: string, isShown: boolean): void => {
    const next = new Set(hiddenSources.value)
    if (isShown) next.delete(source)
    else next.add(source)
    hiddenSources.value = next
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
      items,
      filter: readonly(filter),
      hiddenSources: readonly(hiddenSources),
      failure: readonly(failure),
      isEditing: readonly(isEditing),
      queryProblems,
      functions: readonly(functions),
      getNow: deps.getNow,
      actions: {
        openResult,
        saveSearch,
        setEditing,
        setFilter: (next: string) => {
          filter.value = next
        },
        setSourceShown,
        showAllSources: () => {
          hiddenSources.value = new Set()
        },
      },
    },
    follow,
    dispose,
  }
}
