/**
 * The JSON bodies of the methods the plugin's backend serves under the service `numen.pdf`.
 */

/** The identity of the bytes of a file. */
export interface FileStat {
  readonly path: string
  readonly size: number
  readonly mtime: number
}

/** A run of a text version, as byte offsets from `start` up to `end`. */
export interface TextRun {
  readonly start: number
  readonly end: number
}

/** The size of one page. */
export interface PageSize {
  readonly width: number
  readonly height: number
}

export interface GetDocumentRequest {
  readonly path: string
}

export interface GetDocumentResponse {
  readonly pages: readonly PageSize[]
  readonly stat?: FileStat
}

export interface GetHighlightsRequest {
  readonly path: string
  readonly ranges: readonly TextRun[]
}

/** A rectangle on a page, in fractions of the page size. */
export interface DocumentRect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** The rectangles of a run on one page. */
export interface DocumentPlace {
  readonly page: number
  readonly rects: readonly DocumentRect[]
}

/** Where one range of text is placed, one place for each page. */
export interface DocumentRun {
  readonly places: readonly DocumentPlace[]
}

/** One run for each requested range, in request order. */
export interface GetHighlightsResponse {
  readonly runs: readonly DocumentRun[]
}
