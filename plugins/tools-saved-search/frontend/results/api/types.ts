/**
 * The messages of the saved search service, as the Go structs of backend/api encode them.
 */

/** The place of a result in its source: a page, a time of a recording, a position in a book. */
export interface Place {
  readonly page?: number
  readonly positionSeconds?: number
  readonly endSeconds?: number
  readonly cfi?: string
  readonly chapter?: string
}

/** One result: a passage of a file, or a whole file when isWholeFile is set. */
export interface SearchResult {
  readonly path: string
  readonly title: string
  readonly kind: string
  readonly place?: Place
  readonly text?: string
  readonly start: number
  readonly length: number
  readonly line: number
  readonly uri: string
  readonly score?: number
  readonly isWholeFile?: boolean
}

/**
 * A problem of the query, at its line and column counted from 1 over the query: a problem of the note
 * outside its query at line 1, column 1, a problem of a provider at 0, 0.
 */
export interface Problem {
  readonly line: number
  readonly column: number
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
