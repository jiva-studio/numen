/**
 * What the tab draws for an answer of the service: the state of the run with its details and the
 * problems of the note, the warnings, the results with their places, the sources the results come from,
 * the words for no result; and the results a filter text and the hidden sources leave, and why none is
 * left.
 */
import type { Words } from '../../words'
import { formatPlace, type PlaceFields } from './place'
import { getProblemText } from './problemText'
import { formatTime } from './time'

/** The fields of a result that the tab draws. */
export interface ResultFields {
  readonly path: string
  readonly title: string
  readonly kind: string
  readonly place?: PlaceFields
  readonly text?: string
  readonly isWholeFile?: boolean
}

/** The fields of an answer of GetResults that the tab draws. */
export interface ResultsFields {
  readonly title: string
  readonly query: string
  readonly computedAt: string
  readonly failedAt: string
  readonly failure: string
  readonly isStale: boolean
  readonly isRunning: boolean
  readonly results: readonly ResultFields[]
  readonly warnings: readonly string[]
  readonly errors: readonly {
    readonly line: number
    readonly column: number
    readonly message: string
  }[]
}

/**
 * One result as the tab draws it: index is its place in the answer, source the path of its file, path
 * '' when the title is the file name. A whole file has no place and no text and shows its kind.
 */
export interface ResultItemView {
  readonly index: number
  readonly source: string
  readonly title: string
  readonly path: string
  readonly place: string
  readonly text: string
  readonly kind: string
  readonly isWholeFile: boolean
}

/** A file the results come from, with the title of its first result. */
export interface SourceView {
  readonly path: string
  readonly title: string
}

/** The state of the run: fresh results, a run in progress, results that will update, a failed last run. */
export type RunState = 'fresh' | 'running' | 'stale' | 'failed'

/** The state of the run, its name, and the lines that describe it: the state, the times and the failure. */
export interface StatusView {
  readonly state: RunState
  readonly label: string
  readonly details: readonly string[]
}

/** The tab of a saved search as it is drawn. */
export interface ResultsView {
  readonly title: string
  readonly query: string
  readonly status: StatusView
  readonly warnings: readonly string[]
  readonly items: readonly ResultItemView[]
  readonly sources: readonly SourceView[]
  readonly emptyText: string
}

type Problem = ResultsFields['errors'][number]

/** As `9:4 message`: the line and the column, counted from 1, then the message. */
export function formatProblem(problem: Problem): string {
  return `${problem.line}:${problem.column} ${problem.message}`
}

const formatPlainProblem = (problem: Problem): string =>
  formatProblem({ ...problem, message: getProblemText(problem.message) })

function getFileName(path: string): string {
  return path.slice(path.lastIndexOf('/') + 1)
}

function getFileStem(path: string): string {
  const name = getFileName(path)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(0, dot) : name
}

function isLastRunFailed(response: ResultsFields): boolean {
  if (response.failedAt === '') return false
  if (response.computedAt === '') return true
  return Date.parse(response.failedAt) >= Date.parse(response.computedAt)
}

function getRunState(response: ResultsFields): RunState {
  if (response.isRunning) return 'running'
  if (isLastRunFailed(response)) return 'failed'
  if (response.computedAt === '') return 'running'
  if (response.isStale) return 'stale'
  return 'fresh'
}

function getLabel(state: RunState, response: ResultsFields, words: Words): string {
  if (state === 'running') return response.computedAt === '' ? words.computing : words.updating
  if (state === 'failed') return words.lastRunFailed
  if (state === 'stale') return words.willUpdate
  return words.upToDate
}

function getStatus(response: ResultsFields, words: Words, locale: string): StatusView {
  const state = getRunState(response)
  const label = getLabel(state, response, words)
  const failedRun = isLastRunFailed(response)
    ? `${words.lastRunFailed} ${formatTime(response.failedAt, locale)}: ${response.failure}`
    : ''
  const details = [state === 'failed' ? failedRun : label]
  if (response.computedAt !== '')
    details.push(`${words.computed} ${formatTime(response.computedAt, locale)}`)
  if (response.isStale && state !== 'stale') details.push(words.willUpdate)
  if (failedRun !== '' && state !== 'failed') details.push(failedRun)
  details.push(...response.errors.map(formatPlainProblem))
  return { state, label, details }
}

function getItem(result: ResultFields, index: number, words: Words): ResultItemView {
  const isWholeFile = result.isWholeFile === true
  return {
    index,
    source: result.path,
    title: result.title,
    path: !isWholeFile && result.title === getFileStem(result.path) ? '' : result.path,
    place: isWholeFile ? '' : formatPlace(result.place, words),
    text: isWholeFile ? '' : (result.text ?? ''),
    kind: result.kind,
    isWholeFile,
  }
}

function getSources(results: readonly ResultFields[]): SourceView[] {
  const sources = new Map<string, SourceView>()
  for (const result of results) {
    if (!sources.has(result.path))
      sources.set(result.path, { path: result.path, title: result.title })
  }
  return [...sources.values()]
}

/** Computes what the tab draws for the answer, with the words and the locale of the person. */
export function getResultsView(response: ResultsFields, words: Words, locale: string): ResultsView {
  const isEmpty = response.computedAt !== '' && response.results.length === 0
  return {
    title: response.title,
    query: response.query,
    status: getStatus(response, words, locale),
    warnings: [...response.warnings],
    items: response.results.map((result, index) => getItem(result, index, words)),
    sources: getSources(response.results),
    emptyText: isEmpty ? words.noMatch : '',
  }
}

function isMatch(item: ResultItemView, needle: string): boolean {
  return [item.text, item.title, item.place, getFileName(item.source)].some((field) =>
    field.toLocaleLowerCase().includes(needle),
  )
}

/**
 * The items whose source is not hidden and whose text, title, place or file name holds the filter text,
 * without regard to case; every item of a shown source for an empty filter.
 */
export function filterItems(
  items: readonly ResultItemView[],
  filter: string,
  hidden: ReadonlySet<string>,
): ResultItemView[] {
  const needle = filter.trim().toLocaleLowerCase()
  return items.filter(
    (item) => !hidden.has(item.source) && (needle === '' || isMatch(item, needle)),
  )
}

/** What empties the list of results: every source hidden, or a filter text that no result holds. */
export type EmptyReason = 'sources' | 'filter'

/** Why the filter leaves no result, in words, and the action that brings the results back. */
export interface FilterEmptyView {
  readonly reason: EmptyReason
  readonly text: string
  readonly actionText: string
}

/** Why the shown sources and the filter text leave no result of the sources. */
export function getFilterEmpty(
  sources: readonly SourceView[],
  hidden: ReadonlySet<string>,
  filter: string,
  words: Words,
): FilterEmptyView {
  if (sources.every((source) => hidden.has(source.path)))
    return { reason: 'sources', text: words.allSourcesHidden, actionText: words.showAllSources }
  return {
    reason: 'filter',
    text: words.noFilterMatch(filter.trim()),
    actionText: words.clearFilter,
  }
}
