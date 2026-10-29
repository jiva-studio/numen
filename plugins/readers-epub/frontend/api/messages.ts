/**
 * The JSON messages of the numen.epub service, as the plugin's backend serves them. The shapes
 * match backend/api/api.go and the fixtures in testdata/api.
 */

/** The bytes of a file. `mtime` is the modification time in nanoseconds as a decimal string. */
export interface FileStat {
  path: string
  size: number
  mtime: string
}

export interface GetBookRequest {
  path: string
}

export interface SpineDocument {
  path: string
  offset: number
  length: number
}

export interface BookPart {
  title: string
  offset: number
  level: number
}

export interface PrintedPage {
  label: string
  offset: number
}

export interface GetBookResponse {
  title: string
  documents: SpineDocument[]
  parts: BookPart[]
  printedPages: PrintedPage[]
  pageCount: number
  textBytes: number
  pageBytes: number
  stat?: FileStat
}

export interface ReadBookMarkupRequest {
  path: string
  document: string
  baseline?: FileStat
}

export interface ReadBookMarkupResponse {
  markup: string
}

export interface ReadBookEntryRequest {
  path: string
  entry: string
  baseline?: FileStat
}

/** `body` is the bytes of the entry as base64 text. */
export interface ReadBookEntryResponse {
  body: string
}

export interface GetCfiRequest {
  path: string
  offset: number
}

export interface GetCfiResponse {
  cfi: string
}

export interface FindOffsetRequest {
  path: string
  cfi: string
}

export interface FindOffsetResponse {
  offset: number
}
