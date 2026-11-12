/**
 * The place another component requested to display: a source and the ranges of its text,
 * counted in bytes. A length of zero names the source and no place inside it.
 */
export interface FocusRequest {
  readonly path: string
  readonly ranges: readonly { readonly start: number; readonly end: number }[]
  readonly tabId: string
}
