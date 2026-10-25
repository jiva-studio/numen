/**
 * A run of text, by where it begins and where it ends.
 *
 * What it counts in is the field carrying it: a palette counts the line it
 * draws in UTF-16 code units, and a call working in a source counts that
 * source's text in bytes.
 */
export interface Span {
  readonly from: number
  readonly to: number
}
