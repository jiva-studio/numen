/**
 * The JSON messages of the templates service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.templates'

/** The error codes of a failed method that this plugin reads. */
export const ErrorCode = {
  invalidArgument: 'invalid_argument',
  notFound: 'not_found',
  failedPrecondition: 'failed_precondition',
} as const

/** Asks for a note made from a template. */
export interface CreateNoteRequest {
  readonly template: string
  readonly folder: string
}

/** Names the note that was made. */
export interface CreateNoteResponse {
  readonly path: string
}

/** The request and response of each method by name. */
export interface Methods {
  CreateNote: { request: CreateNoteRequest; response: CreateNoteResponse }
}
