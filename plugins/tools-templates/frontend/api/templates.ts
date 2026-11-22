/**
 * The JSON messages of the templates service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.templates'

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

/**
 * The fields of a template, in the order they are asked, the etag of the template they were read from, and
 * the name no note of the folder has yet that a note from the template is suggested to be given.
 */
export interface ListFieldsResponse {
  readonly fields: readonly Field[]
  readonly etag: string
  readonly name: string
}

/**
 * Asks for a note made from a template. `name` is what the note is called, its file name without the extension;
 * `values` answers its fields by label and `etag` is the one the fields were listed with.
 */
export interface CreateNoteRequest {
  readonly template: string
  readonly folder: string
  readonly name: string
  readonly values: Readonly<Record<string, string>>
  readonly etag: string
}

/** Names the note that was made; `warning` is what the person should know of it, empty for nothing. */
export interface CreateNoteResponse {
  readonly path: string
  readonly warning: string
}
