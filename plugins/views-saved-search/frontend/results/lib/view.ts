/**
 * What the tab draws for an answer of the service: the status line, the failed last run, the problems,
 * the warnings, the results with their places, and the words for no result.
 */
import type { Words } from '../../words'
import { formatPlace, type PlaceFields } from './place'
import { formatTime } from './time'

/** The fields of a result that the tab draws. */
export interface ResultFields {
  readonly path: string
  readonly title: string
  readonly place?: PlaceFields
  readonly text: string
}

/** The fields of an answer of GetResults that the tab draws. */
export interface ResultsFields {
  readonly title: string
  readonly query: string
  readonly rerank: string
  readonly limit: number
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

/** One result as the tab draws it; path is '' when the title is the file name. */
export interface ResultItemView {
  readonly title: string
  readonly path: string
  readonly place: string
  readonly text: string
}

/** The tab of a saved search as it is drawn. */
export interface ResultsView {
  readonly title: string
  readonly query: string
  readonly rerank: string
  readonly limit: number
  readonly status: string
  readonly failedRun: string
  readonly problems: readonly string[]
  readonly warnings: readonly string[]
  readonly items: readonly ResultItemView[]
  readonly emptyText: string
}

/** As `9:4 message`: the line and the column, counted from 1, then the message. */
export function formatProblem(problem: {
  readonly line: number
  readonly column: number
  readonly message: string
}): string {
  return `${problem.line}:${problem.column} ${problem.message}`
}

function getFileStem(path: string): string {
  const name = path.slice(path.lastIndexOf('/') + 1)
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(0, dot) : name
}

function getStatus(response: ResultsFields, words: Words, locale: string): string {
  if (response.computedAt === '') return words.computing
  const time = formatTime(response.computedAt, locale)
  if (response.isRunning) return `${words.updating}${words.separator}${words.computedLower} ${time}`
  if (response.isStale) return `${words.willUpdate}${words.separator}${words.computedLower} ${time}`
  return `${words.computed} ${time}`
}

function getFailedRun(response: ResultsFields, words: Words, locale: string): string {
  if (response.failedAt === '') return ''
  return `${words.lastRunFailed} ${formatTime(response.failedAt, locale)}: ${response.failure}`
}

function getItem(result: ResultFields, words: Words): ResultItemView {
  return {
    title: result.title,
    path: result.title === getFileStem(result.path) ? '' : result.path,
    place: formatPlace(result.place, words),
    text: result.text,
  }
}

/** Computes what the tab draws for the answer, with the words and the locale of the person. */
export function getResultsView(response: ResultsFields, words: Words, locale: string): ResultsView {
  const problems = response.errors.map(formatProblem)
  const hasProblems = problems.length > 0
  const isEmpty = !hasProblems && response.computedAt !== '' && response.results.length === 0
  return {
    title: response.title,
    query: response.query,
    rerank: response.rerank,
    limit: response.limit,
    status: getStatus(response, words, locale),
    failedRun: getFailedRun(response, words, locale),
    problems,
    warnings: [...response.warnings],
    items: hasProblems ? [] : response.results.map((result) => getItem(result, words)),
    emptyText: isEmpty ? words.noMatch : '',
  }
}
