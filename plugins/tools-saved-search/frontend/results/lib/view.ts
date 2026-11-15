/**
 * What the tab draws for an answer of the service: the status line, the failed last run, the problems,
 * the warnings, the results with their places, the sources the results come from, and the words for no
 * result; and the results a filter text and the hidden sources leave.
 */
import type { Words } from '../../words'
import { formatPlace, type PlaceFields } from './place'
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

/** The tab of a saved search as it is drawn. */
export interface ResultsView {
  readonly title: string
  readonly query: string
  readonly status: string
  readonly failedRun: string
  readonly problems: readonly string[]
  readonly warnings: readonly string[]
  readonly items: readonly ResultItemView[]
  readonly sources: readonly SourceView[]
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

function getFileName(path: string): string {
  return path.slice(path.lastIndexOf('/') + 1)
}

function getFileStem(path: string): string {
  const name = getFileName(path)
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
    failedRun: getFailedRun(response, words, locale),
    problems: response.errors.map(formatProblem),
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
