/**
 * The prose around the part a card tab edits, preserved unchanged. A note
 * belongs to its user, and the text above the first card and below the last is
 * written back as it was read.
 */
export interface Surrounds {
  /** The prose below the frontmatter and above the first of them. */
  readonly preamble: string
  /** What the file ends with once the last of them has been read. */
  readonly tail: string
}
