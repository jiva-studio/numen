/**
 * The JSON messages of the importer service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.media.importer'

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
}

interface CopyMediaRequest {
  readonly path: string
  readonly kind: 'video' | 'audio'
}

type CopyMediaResponse = Record<string, never>

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
  CopyMedia: { request: CopyMediaRequest; response: CopyMediaResponse }
  CreateLinks: { request: CreateLinksRequest; response: CreateLinksResponse }
}
