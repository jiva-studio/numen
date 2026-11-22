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

/** Asks what a template asks of the person who makes a note from it. */
export interface ListFieldsRequest {
  readonly template: string
  readonly folder: string
}

/** A question of a template: a line of text, or one of the options. */
export interface Field {
  readonly kind: 'text' | 'select'
  readonly label: string
  readonly options: readonly string[]
  readonly default: string
}

/** The fields of a template, in the order they are asked. */
export interface ListFieldsResponse {
  readonly fields: readonly Field[]
}

/** Asks for a note made from a template; `values` answers its fields by label. */
export interface CreateNoteRequest {
  readonly template: string
  readonly folder: string
  readonly values: Readonly<Record<string, string>>
}

/** Names the note that was made. */
export interface CreateNoteResponse {
  readonly path: string
}

/** The request and response of each method by name. */
export interface Methods {
  ListFields: { request: ListFieldsRequest; response: ListFieldsResponse }
  CreateNote: { request: CreateNoteRequest; response: CreateNoteResponse }
}
