/**
 * The prose standing around what a card tab edits, kept as the person left it.
 * A note is theirs, and what they wrote above the first card and below the last
 * comes back written as it went out.
 */
export interface Surrounds {
  /** The prose below the frontmatter and above the first of them. */
  readonly preamble: string
  /** What the file ends with once the last of them has been read. */
  readonly tail: string
}
