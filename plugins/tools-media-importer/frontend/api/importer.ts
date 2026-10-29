/**
 * The JSON messages of the importer service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

/** The service name the manifest declares. */
export const SERVICE = 'numen.importer'

/** The error codes of a failed method that this plugin reads. */
export const ErrorCode = {
  notFound: 'not_found',
  failedPrecondition: 'failed_precondition',
} as const

export interface GetLinkRequest {
  readonly path: string
}

export interface GetLinkResponse {
  readonly url: string
  readonly isCopyable: boolean
  /** The byte count of the stored copy, zero where none is kept. */
  readonly copySize: number
  readonly isTextDeleted: boolean
  readonly embedPage: string
}

export interface ReadArticleRequest {
  readonly path: string
}

export interface ReadArticleResponse {
  readonly text: string
}

export interface FetchTextRequest {
  readonly path: string
  readonly isUserStarted: boolean
}

export type FetchTextResponse = Record<string, never>

export interface CopyVideoRequest {
  readonly path: string
}

export type CopyVideoResponse = Record<string, never>

export interface DeleteCopyRequest {
  readonly path: string
}

export type DeleteCopyResponse = Record<string, never>

export interface CreateLinksRequest {
  readonly folder: string
  readonly urls: readonly string[]
}

export interface CreateLinksResponse {
  readonly paths: readonly string[]
}

/** The request and response of each method by name. */
export interface Methods {
  GetLink: { request: GetLinkRequest; response: GetLinkResponse }
  ReadArticle: { request: ReadArticleRequest; response: ReadArticleResponse }
  FetchText: { request: FetchTextRequest; response: FetchTextResponse }
  CopyVideo: { request: CopyVideoRequest; response: CopyVideoResponse }
  DeleteCopy: { request: DeleteCopyRequest; response: DeleteCopyResponse }
  CreateLinks: { request: CreateLinksRequest; response: CreateLinksResponse }
}
