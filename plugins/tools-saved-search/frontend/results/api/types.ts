/**
 * The messages of the saved search service, as the Go structs of backend/api encode them.
 */
import type { SearchResult } from './searchResult'

/**
 * A problem of the query, at its line and column counted from 1 over the query: a problem of the note
 * outside its query at line 1, column 1, a problem of a provider at 0, 0. endLine and endColumn are the
 * last character of a problem over more than one.
 */
interface Problem {
  readonly line: number
  readonly column: number
  readonly endLine?: number
  readonly endColumn?: number
  readonly message: string
}

export interface GetResultsRequest {
  readonly path: string
}

/**
 * A saved search: the note as it reads now and the results of its last good run. computedAt is the time
 * of the last good run and failedAt of a run that failed after it, each '' when there is none.
 */
export interface GetResultsResponse {
  readonly path: string
  readonly title: string
  readonly query: string
  readonly computedAt: string
  readonly failedAt: string
  readonly failure: string
  readonly isStale: boolean
  readonly isRunning: boolean
  readonly order: '' | 'reranked' | 'fused'
  readonly results: readonly SearchResult[]
  readonly warnings: readonly string[]
  readonly errors: readonly Problem[]
}

export interface WatchResultsRequest {
  readonly path: string
}

/** A notice that the results of the note changed or a run of it started. */
export interface ResultsChange {
  readonly computedAt: string
  readonly isRunning: boolean
}

/** The query written into the note. */
export interface UpdateSearchRequest {
  readonly path: string
  readonly query: string
}

/** The problems of a query that was not written, counted from its first line; empty when written. */
export interface UpdateSearchResponse {
  readonly errors: readonly Problem[]
}

export type ListSearchFunctionsRequest = Readonly<Record<string, never>>

/** A function, a variable or a stage of the search language; kind is function, variable or stage. */
export interface SearchFunction {
  readonly name: string
  readonly kind: string
  readonly summary: string
  readonly description: string
  readonly example: string
  readonly signatures: readonly string[]
}

/** The catalog of the search language the core runs, with the functions plugins add to it. */
export interface ListSearchFunctionsResponse {
  readonly functions: readonly SearchFunction[]
}
