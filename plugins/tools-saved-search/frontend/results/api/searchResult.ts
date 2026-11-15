/** The place of a result in its source: a page, a time of a recording, a position in a book. */
interface Place {
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
