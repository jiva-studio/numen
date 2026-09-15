/**
 * A run of text, by where it begins and where it ends. What it counts in is the
 * field carrying it.
 */
export interface Span {
  readonly from: number
  readonly to: number
}
